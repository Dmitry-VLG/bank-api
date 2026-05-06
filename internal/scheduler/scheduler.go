package scheduler

import (
	"context"
	"time"

	"bank-api/internal/service"

	"github.com/sirupsen/logrus"
)

type Scheduler struct {
	bank     *service.BankService
	interval time.Duration
	log      *logrus.Logger
}

func New(bank *service.BankService, interval time.Duration, log *logrus.Logger) *Scheduler {
	return &Scheduler{
		bank:     bank,
		interval: interval,
		log:      log,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.process(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.process(ctx)
		}
	}
}

func (s *Scheduler) process(ctx context.Context) {
	processed, err := s.bank.ProcessDuePayments(ctx)
	if err != nil {
		s.log.WithError(err).Error("scheduled payment processing failed")
		return
	}

	s.log.WithField("processed", processed).Info("scheduled payments processed")
}
