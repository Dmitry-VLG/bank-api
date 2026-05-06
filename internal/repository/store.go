package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"bank-api/internal/model"

	"github.com/lib/pq"
)

type Store struct {
	db         *sql.DB
	cardPGPKey string
}

type ScheduleItem struct {
	DueDate     time.Time
	AmountCents int64
}

func NewStore(db *sql.DB, cardPGPKey string) *Store {
	return &Store{
		db:         db,
		cardPGPKey: cardPGPKey,
	}
}

func (s *Store) CreateUser(ctx context.Context, email, username, passwordHash string) (model.User, error) {
	var u model.User

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO users(email, username, password_hash)
		VALUES($1, $2, $3)
		RETURNING id, email, username, password_hash, created_at
	`, email, username, passwordHash).Scan(
		&u.ID,
		&u.Email,
		&u.Username,
		&u.PasswordHash,
		&u.CreatedAt,
	)

	if isUniqueViolation(err) {
		return model.User{}, ErrConflict
	}

	return u, err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User

	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, username, password_hash, created_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&u.ID,
		&u.Email,
		&u.Username,
		&u.PasswordHash,
		&u.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrNotFound
	}

	return u, err
}

func (s *Store) CreateAccount(ctx context.Context, userID int64) (model.Account, error) {
	var a model.Account

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO accounts(user_id, currency)
		VALUES($1, 'RUB')
		RETURNING id, user_id, currency, balance_cents, created_at
	`, userID).Scan(
		&a.ID,
		&a.UserID,
		&a.Currency,
		&a.BalanceCents,
		&a.CreatedAt,
	)

	return model.HydrateAccount(a), err
}

func (s *Store) AccountsByUser(ctx context.Context, userID int64) ([]model.Account, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, currency, balance_cents, created_at
		FROM accounts
		WHERE user_id = $1
		ORDER BY id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.Account

	for rows.Next() {
		var a model.Account

		if err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.Currency,
			&a.BalanceCents,
			&a.CreatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, model.HydrateAccount(a))
	}

	return items, rows.Err()
}

func (s *Store) Deposit(ctx context.Context, userID, accountID, amountCents int64) (model.Account, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Account{}, err
	}
	defer rollback(tx)

	a, err := lockAccountForUser(ctx, tx, userID, accountID)
	if err != nil {
		return model.Account{}, err
	}

	a.BalanceCents += amountCents

	if _, err := tx.ExecContext(ctx, `
		UPDATE accounts
		SET balance_cents = $1
		WHERE id = $2
	`, a.BalanceCents, accountID); err != nil {
		return model.Account{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO transactions(user_id, to_account_id, type, amount_cents)
		VALUES($1, $2, 'deposit', $3)
	`, userID, accountID, amountCents); err != nil {
		return model.Account{}, err
	}

	return model.HydrateAccount(a), tx.Commit()
}

func (s *Store) Withdraw(ctx context.Context, userID, accountID, amountCents int64) (model.Account, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Account{}, err
	}
	defer rollback(tx)

	a, err := lockAccountForUser(ctx, tx, userID, accountID)
	if err != nil {
		return model.Account{}, err
	}

	if a.BalanceCents < amountCents {
		return model.Account{}, ErrInsufficientFunds
	}

	a.BalanceCents -= amountCents

	if _, err := tx.ExecContext(ctx, `
		UPDATE accounts
		SET balance_cents = $1
		WHERE id = $2
	`, a.BalanceCents, accountID); err != nil {
		return model.Account{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO transactions(user_id, from_account_id, type, amount_cents)
		VALUES($1, $2, 'withdrawal', $3)
	`, userID, accountID, amountCents); err != nil {
		return model.Account{}, err
	}

	return model.HydrateAccount(a), tx.Commit()
}

func (s *Store) Transfer(ctx context.Context, userID, fromAccountID, toAccountID, amountCents int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

	from, err := lockAccountForUser(ctx, tx, userID, fromAccountID)
	if err != nil {
		return err
	}

	if from.BalanceCents < amountCents {
		return ErrInsufficientFunds
	}

	var toUserID int64
	var toBalance int64

	err = tx.QueryRowContext(ctx, `
		SELECT user_id, balance_cents
		FROM accounts
		WHERE id = $1
		FOR UPDATE
	`, toAccountID).Scan(&toUserID, &toBalance)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE accounts
		SET balance_cents = balance_cents - $1
		WHERE id = $2
	`, amountCents, fromAccountID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE accounts
		SET balance_cents = balance_cents + $1
		WHERE id = $2
	`, amountCents, toAccountID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO transactions(user_id, from_account_id, to_account_id, type, amount_cents)
		VALUES($1, $2, $3, 'transfer_out', $4)
	`, userID, fromAccountID, toAccountID, amountCents); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO transactions(user_id, from_account_id, to_account_id, type, amount_cents)
		VALUES($1, $2, $3, 'transfer_in', $4)
	`, toUserID, fromAccountID, toAccountID, amountCents); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) CreateCard(
	ctx context.Context,
	userID int64,
	accountID int64,
	number string,
	expiry string,
	cvvHash string,
	dataHMAC string,
	last4 string,
) (model.Card, error) {
	var c model.Card

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO cards(user_id, account_id, number_encrypted, expiry_encrypted, cvv_hash, data_hmac, last4)
		SELECT $1, id, pgp_sym_encrypt($3, $6), pgp_sym_encrypt($4, $6), $5, $7, $8
		FROM accounts
		WHERE id = $2 AND user_id = $1
		RETURNING id, user_id, account_id, last4, created_at
	`, userID, accountID, number, expiry, cvvHash, s.cardPGPKey, dataHMAC, last4).Scan(
		&c.ID,
		&c.UserID,
		&c.AccountID,
		&c.Last4,
		&c.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return model.Card{}, ErrForbidden
	}

	return c, err
}

func (s *Store) CardsByUser(ctx context.Context, userID int64) ([]model.Card, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, account_id,
		       pgp_sym_decrypt(number_encrypted, $2),
		       pgp_sym_decrypt(expiry_encrypted, $2),
		       data_hmac, last4, created_at
		FROM cards
		WHERE user_id = $1
		ORDER BY id
	`, userID, s.cardPGPKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.Card

	for rows.Next() {
		var c model.Card

		if err := rows.Scan(
			&c.ID,
			&c.UserID,
			&c.AccountID,
			&c.Number,
			&c.Expiry,
			&c.DataHMAC,
			&c.Last4,
			&c.CreatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, c)
	}

	return items, rows.Err()
}
func (s *Store) CardByIDForUser(ctx context.Context, userID, cardID int64) (model.Card, error) {
	var c model.Card

	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, account_id,
		       pgp_sym_decrypt(number_encrypted, $3),
		       pgp_sym_decrypt(expiry_encrypted, $3),
		       data_hmac, last4, created_at
		FROM cards
		WHERE id = $1 AND user_id = $2
	`, cardID, userID, s.cardPGPKey).Scan(
		&c.ID,
		&c.UserID,
		&c.AccountID,
		&c.Number,
		&c.Expiry,
		&c.DataHMAC,
		&c.Last4,
		&c.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return model.Card{}, ErrForbidden
	}

	return c, err
}

func (s *Store) CardPayment(ctx context.Context, userID, cardID, amountCents int64) (model.Account, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Account{}, err
	}
	defer rollback(tx)

	var a model.Account

	err = tx.QueryRowContext(ctx, `
		SELECT a.id, a.user_id, a.currency, a.balance_cents, a.created_at
		FROM cards c
		JOIN accounts a ON a.id = c.account_id
		WHERE c.id = $1 AND c.user_id = $2
		FOR UPDATE OF a
	`, cardID, userID).Scan(
		&a.ID,
		&a.UserID,
		&a.Currency,
		&a.BalanceCents,
		&a.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return model.Account{}, ErrForbidden
	}
	if err != nil {
		return model.Account{}, err
	}

	if a.BalanceCents < amountCents {
		return model.Account{}, ErrInsufficientFunds
	}

	a.BalanceCents -= amountCents

	if _, err := tx.ExecContext(ctx, `
		UPDATE accounts
		SET balance_cents = $1
		WHERE id = $2
	`, a.BalanceCents, a.ID); err != nil {
		return model.Account{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO transactions(user_id, from_account_id, type, amount_cents)
		VALUES($1, $2, 'card_payment', $3)
	`, userID, a.ID, amountCents); err != nil {
		return model.Account{}, err
	}

	return model.HydrateAccount(a), tx.Commit()
}

func (s *Store) UserEmailByID(ctx context.Context, userID int64) (string, error) {
	var email string
	err := s.db.QueryRowContext(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return email, err
}

func (s *Store) CreateCreditWithSchedule(
	ctx context.Context,
	userID int64,
	accountID int64,
	principalCents int64,
	annualRate float64,
	termMonths int,
	monthlyPaymentCents int64,
	schedule []ScheduleItem,
) (model.Credit, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Credit{}, err
	}
	defer rollback(tx)

	if _, err := lockAccountForUser(ctx, tx, userID, accountID); err != nil {
		return model.Credit{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE accounts
		SET balance_cents = balance_cents + $1
		WHERE id = $2
	`, principalCents, accountID); err != nil {
		return model.Credit{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO transactions(user_id, to_account_id, type, amount_cents)
		VALUES($1, $2, 'credit_disbursement', $3)
	`, userID, accountID, principalCents); err != nil {
		return model.Credit{}, err
	}

	var c model.Credit

	err = tx.QueryRowContext(ctx, `
		INSERT INTO credits(user_id, account_id, principal_cents, annual_rate, term_months, monthly_payment_cents)
		VALUES($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, account_id, principal_cents, annual_rate, term_months, monthly_payment_cents, status, created_at
	`, userID, accountID, principalCents, annualRate, termMonths, monthlyPaymentCents).Scan(
		&c.ID,
		&c.UserID,
		&c.AccountID,
		&c.PrincipalCents,
		&c.AnnualRate,
		&c.TermMonths,
		&c.MonthlyPaymentCents,
		&c.Status,
		&c.CreatedAt,
	)
	if err != nil {
		return model.Credit{}, err
	}

	for _, item := range schedule {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO payment_schedules(credit_id, account_id, due_date, amount_cents)
			VALUES($1, $2, $3, $4)
		`, c.ID, accountID, item.DueDate, item.AmountCents); err != nil {
			return model.Credit{}, err
		}
	}

	return model.HydrateCredit(c), tx.Commit()
}

func (s *Store) PaymentSchedule(ctx context.Context, userID, creditID int64) ([]model.PaymentSchedule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ps.id, ps.credit_id, ps.account_id, ps.due_date, ps.amount_cents, ps.penalty_cents, ps.status, ps.paid_at
		FROM payment_schedules ps
		JOIN credits c ON c.id = ps.credit_id
		WHERE ps.credit_id = $1 AND c.user_id = $2
		ORDER BY ps.due_date
	`, creditID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSchedules(rows)
}

func (s *Store) Analytics(ctx context.Context, userID int64) (model.Analytics, error) {
	var a model.Analytics

	err := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(amount_cents) FILTER (WHERE type IN ('deposit', 'transfer_in', 'credit_disbursement')), 0),
			COALESCE(SUM(amount_cents) FILTER (WHERE type IN ('withdrawal', 'transfer_out', 'credit_payment', 'card_payment')), 0)
		FROM transactions
		WHERE user_id = $1
		  AND created_at >= date_trunc('month', now())
		  AND created_at < date_trunc('month', now()) + interval '1 month'
	`, userID).Scan(&a.MonthIncomeCents, &a.MonthExpenseCents)
	if err != nil {
		return model.Analytics{}, err
	}

	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(ps.amount_cents + ps.penalty_cents), 0)
		FROM payment_schedules ps
		JOIN credits c ON c.id = ps.credit_id
		WHERE c.user_id = $1 AND ps.status IN ('pending', 'overdue')
	`, userID).Scan(&a.CreditLoadCents)
	if err != nil {
		return model.Analytics{}, err
	}

	a.MonthIncome = model.FromCents(a.MonthIncomeCents)
	a.MonthExpense = model.FromCents(a.MonthExpenseCents)
	a.CreditLoad = model.FromCents(a.CreditLoadCents)

	return a, nil
}

func (s *Store) PredictBalance(ctx context.Context, userID, accountID int64, days int) (model.BalancePrediction, error) {
	var balance int64

	err := s.db.QueryRowContext(ctx, `
		SELECT balance_cents
		FROM accounts
		WHERE id = $1 AND user_id = $2
	`, accountID, userID).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return model.BalancePrediction{}, ErrNotFound
	}
	if err != nil {
		return model.BalancePrediction{}, err
	}

	var planned int64

	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount_cents + penalty_cents), 0)
		FROM payment_schedules
		WHERE account_id = $1
		  AND status IN ('pending', 'overdue')
		  AND due_date <= CURRENT_DATE + ($2::int * interval '1 day')
	`, accountID, days).Scan(&planned)
	if err != nil {
		return model.BalancePrediction{}, err
	}

	p := model.BalancePrediction{
		AccountID:           accountID,
		Days:                days,
		CurrentBalanceCents: balance,
		PlannedDebitsCents:  planned,
		PredictedCents:      balance - planned,
	}

	p.CurrentBalance = model.FromCents(p.CurrentBalanceCents)
	p.PlannedDebits = model.FromCents(p.PlannedDebitsCents)
	p.Predicted = model.FromCents(p.PredictedCents)

	return p, nil
}

func (s *Store) ProcessDuePayments(ctx context.Context) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer rollback(tx)

	rows, err := tx.QueryContext(ctx, `
		SELECT ps.id, ps.credit_id, ps.account_id, ps.amount_cents, ps.penalty_cents, c.user_id
		FROM payment_schedules ps
		JOIN credits c ON c.id = ps.credit_id
		WHERE ps.status IN ('pending', 'overdue')
		  AND ps.due_date <= CURRENT_DATE
		ORDER BY ps.due_date
		FOR UPDATE OF ps SKIP LOCKED
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type duePayment struct {
		ID           int64
		CreditID     int64
		AccountID    int64
		AmountCents  int64
		PenaltyCents int64
		UserID       int64
	}

	var payments []duePayment

	for rows.Next() {
		var p duePayment

		if err := rows.Scan(
			&p.ID,
			&p.CreditID,
			&p.AccountID,
			&p.AmountCents,
			&p.PenaltyCents,
			&p.UserID,
		); err != nil {
			return 0, err
		}

		payments = append(payments, p)
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	processed := 0

	for _, p := range payments {
		var balance int64

		if err := tx.QueryRowContext(ctx, `
			SELECT balance_cents
			FROM accounts
			WHERE id = $1
			FOR UPDATE
		`, p.AccountID).Scan(&balance); err != nil {
			return 0, err
		}

		total := p.AmountCents + p.PenaltyCents

		if balance >= total {
			if _, err := tx.ExecContext(ctx, `
				UPDATE accounts
				SET balance_cents = balance_cents - $1
				WHERE id = $2
			`, total, p.AccountID); err != nil {
				return 0, err
			}

			if _, err := tx.ExecContext(ctx, `
				UPDATE payment_schedules
				SET status = 'paid', paid_at = now()
				WHERE id = $1
			`, p.ID); err != nil {
				return 0, err
			}

			if _, err := tx.ExecContext(ctx, `
				INSERT INTO transactions(user_id, from_account_id, type, amount_cents)
				VALUES($1, $2, 'credit_payment', $3)
			`, p.UserID, p.AccountID, total); err != nil {
				return 0, err
			}

			processed++
			continue
		}

		penalty := p.PenaltyCents
		if penalty == 0 {
			penalty = p.AmountCents / 10
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE payment_schedules
			SET status = 'overdue', penalty_cents = $2
			WHERE id = $1
		`, p.ID, penalty); err != nil {
			return 0, err
		}
	}

	return processed, tx.Commit()
}

func lockAccountForUser(ctx context.Context, tx *sql.Tx, userID, accountID int64) (model.Account, error) {
	var a model.Account

	err := tx.QueryRowContext(ctx, `
		SELECT id, user_id, currency, balance_cents, created_at
		FROM accounts
		WHERE id = $1
		FOR UPDATE
	`, accountID).Scan(
		&a.ID,
		&a.UserID,
		&a.Currency,
		&a.BalanceCents,
		&a.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return model.Account{}, ErrNotFound
	}
	if err != nil {
		return model.Account{}, err
	}

	if a.UserID != userID {
		return model.Account{}, ErrForbidden
	}

	return a, nil
}

func scanSchedules(rows *sql.Rows) ([]model.PaymentSchedule, error) {
	var items []model.PaymentSchedule

	for rows.Next() {
		var s model.PaymentSchedule
		var paidAt sql.NullTime

		if err := rows.Scan(
			&s.ID,
			&s.CreditID,
			&s.AccountID,
			&s.DueDate,
			&s.AmountCents,
			&s.PenaltyCents,
			&s.Status,
			&paidAt,
		); err != nil {
			return nil, err
		}

		if paidAt.Valid {
			t := paidAt.Time
			s.PaidAt = &t
		}

		items = append(items, model.HydrateSchedule(s))
	}

	return items, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}
