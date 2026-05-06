package service

import (
	"crypto/tls"
	"fmt"

	"bank-api/internal/config"

	"github.com/sirupsen/logrus"
	gomail "gopkg.in/gomail.v2"
)

type EmailSender struct {
	cfg config.Config
	log *logrus.Logger
}

func NewEmailSender(cfg config.Config, log *logrus.Logger) *EmailSender {
	return &EmailSender{
		cfg: cfg,
		log: log,
	}
}

func (s *EmailSender) SendPaymentEmail(to string, amount float64) error {
	if s.cfg.SMTPHost == "" || s.cfg.SMTPUser == "" || s.cfg.SMTPPass == "" {
		s.log.WithField("to", to).Debug("smtp is disabled, skip email")
		return nil
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.cfg.SMTPFrom)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Платеж успешно проведен")
	m.SetBody("text/html", fmt.Sprintf(`
		<h1>Платеж успешно проведен</h1>
		<p>Сумма: <strong>%.2f RUB</strong></p>
		<small>Это автоматическое уведомление.</small>
	`, amount))

	d := gomail.NewDialer(s.cfg.SMTPHost, s.cfg.SMTPPort, s.cfg.SMTPUser, s.cfg.SMTPPass)
	d.TLSConfig = &tls.Config{
		ServerName:         s.cfg.SMTPHost,
		InsecureSkipVerify: false,
	}

	return d.DialAndSend(m)
}
