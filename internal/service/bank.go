package service

import (
	"context"
	"errors"
	"math"
	"time"

	"bank-api/internal/model"
	"bank-api/internal/repository"
	"bank-api/internal/security"
)

type BankService struct {
	store      *repository.Store
	cbr        *CBRClient
	email      *EmailSender
	hmacSecret []byte
}

type AmountInput struct {
	Amount float64 `json:"amount"`
}

type TransferInput struct {
	FromAccountID int64   `json:"from_account_id"`
	ToAccountID   int64   `json:"to_account_id"`
	Amount        float64 `json:"amount"`
}

type CreateCardInput struct {
	AccountID int64 `json:"account_id"`
}

type CreateCreditInput struct {
	AccountID  int64   `json:"account_id"`
	Amount     float64 `json:"amount"`
	TermMonths int     `json:"term_months"`
}

func NewBankService(store *repository.Store, cbr *CBRClient, email *EmailSender, hmacSecret string) *BankService {
	return &BankService{
		store:      store,
		cbr:        cbr,
		email:      email,
		hmacSecret: []byte(hmacSecret),
	}
}

func (s *BankService) CreateAccount(ctx context.Context, userID int64) (model.Account, error) {
	return s.store.CreateAccount(ctx, userID)
}

func (s *BankService) Accounts(ctx context.Context, userID int64) ([]model.Account, error) {
	return s.store.AccountsByUser(ctx, userID)
}

func (s *BankService) Deposit(ctx context.Context, userID, accountID int64, amount float64) (model.Account, error) {
	cents, err := validAmount(amount)
	if err != nil {
		return model.Account{}, err
	}

	a, err := s.store.Deposit(ctx, userID, accountID, cents)
	return a, mapRepoErr(err)
}

func (s *BankService) Withdraw(ctx context.Context, userID, accountID int64, amount float64) (model.Account, error) {
	cents, err := validAmount(amount)
	if err != nil {
		return model.Account{}, err
	}

	a, err := s.store.Withdraw(ctx, userID, accountID, cents)
	return a, mapRepoErr(err)
}

func (s *BankService) Transfer(ctx context.Context, userID int64, in TransferInput) error {
	cents, err := validAmount(in.Amount)
	if err != nil {
		return err
	}

	if in.FromAccountID <= 0 || in.ToAccountID <= 0 || in.FromAccountID == in.ToAccountID {
		return ErrInvalidInput
	}

	return mapRepoErr(s.store.Transfer(ctx, userID, in.FromAccountID, in.ToAccountID, cents))
}

func (s *BankService) CreateCard(ctx context.Context, userID int64, in CreateCardInput) (model.Card, error) {
	if in.AccountID <= 0 {
		return model.Card{}, ErrInvalidInput
	}

	card, err := security.GenerateCard()
	if err != nil {
		return model.Card{}, err
	}

	cvvHash, err := security.HashCVV(card.CVV)
	if err != nil {
		return model.Card{}, err
	}

	h := security.ComputeHMAC(card.Number+"|"+card.Expiry+"|"+card.Last4, s.hmacSecret)

	c, err := s.store.CreateCard(ctx, userID, in.AccountID, card.Number, card.Expiry, cvvHash, h, card.Last4)
	return c, mapRepoErr(err)
}

func (s *BankService) Cards(ctx context.Context, userID int64) ([]model.Card, error) {
	return s.store.CardsByUser(ctx, userID)
}

func (s *BankService) CreateCredit(ctx context.Context, userID int64, in CreateCreditInput) (model.Credit, error) {
	cents, err := validAmount(in.Amount)
	if err != nil {
		return model.Credit{}, err
	}

	if in.AccountID <= 0 || in.TermMonths <= 0 || in.TermMonths > 360 {
		return model.Credit{}, ErrInvalidInput
	}

	annualRate, err := s.cbr.KeyRate(ctx)
	if err != nil {
		return model.Credit{}, err
	}

	monthlyPayment := annuityPayment(cents, annualRate, in.TermMonths)
	if monthlyPayment <= 0 {
		return model.Credit{}, ErrInvalidInput
	}

	schedule := make([]repository.ScheduleItem, 0, in.TermMonths)
	now := time.Now()

	for i := 1; i <= in.TermMonths; i++ {
		schedule = append(schedule, repository.ScheduleItem{
			DueDate:     now.AddDate(0, i, 0),
			AmountCents: monthlyPayment,
		})
	}

	c, err := s.store.CreateCreditWithSchedule(
		ctx,
		userID,
		in.AccountID,
		cents,
		annualRate,
		in.TermMonths,
		monthlyPayment,
		schedule,
	)

	return c, mapRepoErr(err)
}

func (s *BankService) CreditSchedule(ctx context.Context, userID, creditID int64) ([]model.PaymentSchedule, error) {
	if creditID <= 0 {
		return nil, ErrInvalidInput
	}

	items, err := s.store.PaymentSchedule(ctx, userID, creditID)
	return items, mapRepoErr(err)
}

func (s *BankService) Analytics(ctx context.Context, userID int64) (model.Analytics, error) {
	return s.store.Analytics(ctx, userID)
}

func (s *BankService) PredictBalance(ctx context.Context, userID, accountID int64, days int) (model.BalancePrediction, error) {
	if accountID <= 0 || days <= 0 || days > 365 {
		return model.BalancePrediction{}, ErrInvalidInput
	}

	p, err := s.store.PredictBalance(ctx, userID, accountID, days)
	return p, mapRepoErr(err)
}

func (s *BankService) ProcessDuePayments(ctx context.Context) (int, error) {
	return s.store.ProcessDuePayments(ctx)
}

func annuityPayment(principalCents int64, annualRate float64, months int) int64 {
	principal := float64(principalCents)
	i := annualRate / 100 / 12
	n := float64(months)

	if i == 0 {
		return int64(math.Ceil(principal / n))
	}

	payment := principal * i * math.Pow(1+i, n) / (math.Pow(1+i, n) - 1)
	return int64(math.Ceil(payment))
}

func validAmount(amount float64) (int64, error) {
	cents := model.ToCents(amount)
	if amount <= 0 || cents <= 0 {
		return 0, ErrInvalidInput
	}

	return cents, nil
}

func mapRepoErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, repository.ErrConflict):
		return ErrConflict
	case errors.Is(err, repository.ErrForbidden):
		return ErrForbidden
	case errors.Is(err, repository.ErrInsufficientFunds):
		return ErrInsufficientFunds
	default:
		return err
	}
}
