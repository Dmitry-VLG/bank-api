package model

import (
	"math"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

const CurrencyRUB = "RUB"

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Account struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Currency     string    `json:"currency"`
	BalanceCents int64     `json:"balance_cents"`
	Balance      float64   `json:"balance"`
	CreatedAt    time.Time `json:"created_at"`
}

type Card struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	AccountID int64     `json:"account_id"`
	Number    string    `json:"number,omitempty"`
	Expiry    string    `json:"expiry,omitempty"`
	Last4     string    `json:"last4"`
	CreatedAt time.Time `json:"created_at"`
}

type Transaction struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	FromAccountID *int64    `json:"from_account_id,omitempty"`
	ToAccountID   *int64    `json:"to_account_id,omitempty"`
	Type          string    `json:"type"`
	AmountCents   int64     `json:"amount_cents"`
	Amount        float64   `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
}

type Credit struct {
	ID                  int64     `json:"id"`
	UserID              int64     `json:"user_id"`
	AccountID           int64     `json:"account_id"`
	PrincipalCents      int64     `json:"principal_cents"`
	Principal           float64   `json:"principal"`
	AnnualRate          float64   `json:"annual_rate"`
	TermMonths          int       `json:"term_months"`
	MonthlyPaymentCents int64     `json:"monthly_payment_cents"`
	MonthlyPayment      float64   `json:"monthly_payment"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
}

type PaymentSchedule struct {
	ID           int64      `json:"id"`
	CreditID     int64      `json:"credit_id"`
	AccountID    int64      `json:"account_id"`
	DueDate      time.Time  `json:"due_date"`
	AmountCents  int64      `json:"amount_cents"`
	Amount       float64    `json:"amount"`
	PenaltyCents int64      `json:"penalty_cents"`
	Penalty      float64    `json:"penalty"`
	Status       string     `json:"status"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
}

type Analytics struct {
	MonthIncomeCents  int64   `json:"month_income_cents"`
	MonthIncome       float64 `json:"month_income"`
	MonthExpenseCents int64   `json:"month_expense_cents"`
	MonthExpense      float64 `json:"month_expense"`
	CreditLoadCents   int64   `json:"credit_load_cents"`
	CreditLoad        float64 `json:"credit_load"`
}

type BalancePrediction struct {
	AccountID           int64   `json:"account_id"`
	Days                int     `json:"days"`
	CurrentBalanceCents int64   `json:"current_balance_cents"`
	CurrentBalance      float64 `json:"current_balance"`
	PlannedDebitsCents  int64   `json:"planned_debits_cents"`
	PlannedDebits       float64 `json:"planned_debits"`
	PredictedCents      int64   `json:"predicted_cents"`
	Predicted           float64 `json:"predicted"`
}

func ValidateEmail(email string) bool {
	email = strings.TrimSpace(email)
	_, err := mail.ParseAddress(email)
	return err == nil
}

func ValidateUsername(username string) bool {
	return usernameRe.MatchString(username)
}

func ValidatePassword(password string) bool {
	return len(password) >= 8
}

func ToCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

func FromCents(cents int64) float64 {
	return float64(cents) / 100
}

func HydrateAccount(a Account) Account {
	a.Balance = FromCents(a.BalanceCents)
	return a
}

func HydrateCredit(c Credit) Credit {
	c.Principal = FromCents(c.PrincipalCents)
	c.MonthlyPayment = FromCents(c.MonthlyPaymentCents)
	return c
}

func HydrateSchedule(s PaymentSchedule) PaymentSchedule {
	s.Amount = FromCents(s.AmountCents)
	s.Penalty = FromCents(s.PenaltyCents)
	return s
}
