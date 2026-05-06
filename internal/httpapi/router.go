package httpapi

import (
	"net/http"

	"bank-api/internal/security"

	"github.com/gorilla/mux"
)

func NewRouter(h *Handler, tokens security.TokenManager) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/register", h.Register).Methods(http.MethodPost)
	r.HandleFunc("/login", h.Login).Methods(http.MethodPost)

	auth := r.PathPrefix("/").Subrouter()
	auth.Use(AuthMiddleware(tokens))

	auth.HandleFunc("/accounts", h.CreateAccount).Methods(http.MethodPost)
	auth.HandleFunc("/accounts", h.Accounts).Methods(http.MethodGet)
	auth.HandleFunc("/accounts/{accountId:[0-9]+}/deposit", h.Deposit).Methods(http.MethodPost)
	auth.HandleFunc("/accounts/{accountId:[0-9]+}/withdraw", h.Withdraw).Methods(http.MethodPost)
	auth.HandleFunc("/accounts/{accountId:[0-9]+}/predict", h.PredictBalance).Methods(http.MethodGet)

	auth.HandleFunc("/transfer", h.Transfer).Methods(http.MethodPost)

	auth.HandleFunc("/cards", h.CreateCard).Methods(http.MethodPost)
	auth.HandleFunc("/cards", h.Cards).Methods(http.MethodGet)

	auth.HandleFunc("/credits", h.CreateCredit).Methods(http.MethodPost)
	auth.HandleFunc("/credits/{creditId:[0-9]+}/schedule", h.CreditSchedule).Methods(http.MethodGet)

	auth.HandleFunc("/analytics", h.Analytics).Methods(http.MethodGet)

	return r
}
