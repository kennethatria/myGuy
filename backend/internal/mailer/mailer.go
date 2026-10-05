// Package mailer delivers login codes by email.
package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

const (
	dialTimeout    = 10 * time.Second
	sessionTimeout = 30 * time.Second
)

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string // e.g. "MyGuy <no-reply@myguy.work>"
}

// SMTPSender sends mail over SMTP with mandatory STARTTLS, so credentials and
// codes never cross the network in clear text.
type SMTPSender struct {
	cfg SMTPConfig
}

func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) SendLoginCode(ctx context.Context, email, code string) error {
	from, err := mail.ParseAddress(s.cfg.From)
	if err != nil {
		return fmt.Errorf("invalid SMTP_FROM: %w", err)
	}
	to, err := recipient(email)
	if err != nil {
		return err
	}
	msg := buildLoginCodeMessage(from, to, code, time.Now())

	// net/smtp has no timeouts of its own; bound dial and session explicitly.
	dialer := net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(s.cfg.Host, s.cfg.Port))
	if err != nil {
		return err
	}
	if err := conn.SetDeadline(time.Now().Add(sessionTimeout)); err != nil {
		conn.Close()
		return err
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return errors.New("SMTP server does not support STARTTLS")
	}
	if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
		return err
	}
	if s.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(from.Address); err != nil {
		return err
	}
	if err := client.Rcpt(to.Address); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

var errInvalidRecipient = errors.New("invalid recipient: expected a bare email address")

// recipient accepts only a bare address ("a@b.com"): no display name, no
// line breaks. The address comes from the sign-in request and is written
// into the To header, so anything else could inject headers.
func recipient(email string) (*mail.Address, error) {
	if strings.ContainsAny(email, "\r\n") {
		return nil, errInvalidRecipient
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Name != "" || addr.Address != email {
		return nil, errInvalidRecipient
	}
	return &mail.Address{Address: addr.Address}, nil
}

func buildLoginCodeMessage(from, to *mail.Address, code string, now time.Time) []byte {
	headers := []string{
		"From: " + from.String(),
		"To: " + to.String(),
		"Subject: Your MyGuy sign-in code: " + code,
		"Date: " + now.Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}
	body := []string{
		"Your MyGuy sign-in code is:",
		"",
		"    " + code,
		"",
		"It expires in 10 minutes. If you didn't request it, you can ignore this email.",
	}
	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + strings.Join(body, "\r\n") + "\r\n")
}

// LogSender logs codes instead of emailing them. For local development only,
// when no SMTP server is configured.
type LogSender struct{}

func (LogSender) SendLoginCode(_ context.Context, email, code string) error {
	log.Printf("[mailer] SMTP not configured; login code for %s is %s", email, code)
	return nil
}
