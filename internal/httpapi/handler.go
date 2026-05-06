package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"bank-api/internal/model"
	"bank-api/internal/service"

	"github.com/gorilla/mux"
)

type Handler struct {
	auth *service.AuthService
	bank *service.BankService
}

func NewHandler(auth *service.AuthService, bank *service.BankService) *Handler {
	return &Handler{
		auth: auth,
		bank: bank,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in service.RegisterInput

	if !decodeJSON(w, r, &in) {
		return
	}

	resp, err := h.auth.Register(r.Context(), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in service.LoginInput

	if !decodeJSON(w, r, &in) {
		return
	}

	resp, err := h.auth.Login(r.Context(), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}

	a, err := h.bank.CreateAccount(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, a)
}

func (h *Handler) Accounts(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}

	items, err := h.bank.Accounts(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) Deposit(w http.ResponseWriter, r *http.Request) {
	h.amountOperation(w, r, h.bank.Deposit)
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	h.amountOperation(w, r, h.bank.Withdraw)
}

func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}

	var in service.TransferInput

	if !decodeJSON(w, r, &in) {
		return
	}

	if err := h.bank.Transfer(r.Context(), userID, in); err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) CreateCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}

	var in service.CreateCardInput

	if !decodeJSON(w, r, &in) {
		return
	}

	card, err := h.bank.CreateCard(r.Context(), userID, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, card)
}

func (h *Handler) Cards(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}

	cards, err := h.bank.Cards(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, cards)
}

func (h *Handler) CreateCredit(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}

	var in service.CreateCreditInput

	if !decodeJSON(w, r, &in) {
		return
	}

	credit, err := h.bank.CreateCredit(r.Context(), userID, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, credit)
}

func (h *Handler) CreditSchedule(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}

	creditID, err := pathInt64(r, "creditId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid credit id")
		return
	}

	items, err := h.bank.CreditSchedule(r.Context(), userID, creditID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) Analytics(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}

	a, err := h.bank.Analytics(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) PredictBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}

	accountID, err := pathInt64(r, "accountId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	days, err := strconv.Atoi(r.URL.Query().Get("days"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid days")
		return
	}

	p, err := h.bank.PredictBalance(r.Context(), userID, accountID, days)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) amountOperation(
	w http.ResponseWriter,
	r *http.Request,
	fn func(ctx context.Context, userID, accountID int64, amount float64) (model.Account, error),
) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}

	accountID, err := pathInt64(r, "accountId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	var in service.AmountInput

	if !decodeJSON(w, r, &in) {
		return
	}

	resp, err := fn(r.Context(), userID, accountID, in.Amount)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func requireUser(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return 0, false
	}

	return userID, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return false
	}

	return true
}

func pathInt64(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(mux.Vars(r)[name], 10, 64)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInsufficientFunds):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
