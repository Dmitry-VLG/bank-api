package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"bank-api/internal/config"
	"bank-api/internal/db"
	"bank-api/internal/httpapi"
	"bank-api/internal/repository"
	"bank-api/internal/scheduler"
	"bank-api/internal/security"
	"bank-api/internal/service"

	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})

	cfg, err := config.Load()
	if err != nil {
		log.WithError(err).Fatal("config load failed")
	}

	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err == nil {
		log.SetLevel(level)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	conn, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.WithError(err).Fatal("database connection failed")
	}
	defer conn.Close()

	store := repository.NewStore(conn, cfg.CardPGPKey)
	tokens := security.NewTokenManager(cfg.JWTSecret)

	authSvc := service.NewAuthService(store, tokens)
	cbr := service.NewCBRClient(cfg.CBRMarginPercent)
	email := service.NewEmailSender(cfg, log)
	bankSvc := service.NewBankService(store, cbr, email, cfg.CardHMACSecret)

	handler := httpapi.NewHandler(authSvc, bankSvc)
	router := httpapi.NewRouter(handler, tokens)

	sched := scheduler.New(bankSvc, cfg.SchedulerInterval, log)
	go sched.Run(ctx)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.WithField("addr", srv.Addr).Info("http server started")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal("http server failed")
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("http server shutdown failed")
	}
}
