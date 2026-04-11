package smtp

import (
	"context"
	"fmt"

	gomail "github.com/wneessen/go-mail"
	"github.com/semmidev/go-todo-app/internal/port/output"
)

// Config holds SMTP connection parameters loaded from AppConfig.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Sender implements output.EmailSender using go-mail (TLS-aware, context-safe).
type Sender struct {
	cfg    Config
	client *gomail.Client
}

// NewSender creates a ready-to-use SMTP Sender.
// It validates connectivity at startup so wiring failures are caught early.
func NewSender(cfg Config) (*Sender, error) {
	opts := []gomail.Option{
		gomail.WithPort(cfg.Port),
		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
		gomail.WithUsername(cfg.Username),
		gomail.WithPassword(cfg.Password),
	}

	// For local dev (Mailpit) skip TLS; for production enforce it.
	if cfg.Port == 1025 || cfg.Port == 25 {
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	} else {
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	}

	client, err := gomail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("smtp: create client: %w", err)
	}
	return &Sender{cfg: cfg, client: client}, nil
}

// Send delivers a single HTML email. It satisfies output.EmailSender.
func (s *Sender) Send(ctx context.Context, msg output.EmailMessage) error {
	m := gomail.NewMsg()
	if err := m.From(s.cfg.From); err != nil {
		return fmt.Errorf("smtp: set from: %w", err)
	}
	if err := m.To(msg.To); err != nil {
		return fmt.Errorf("smtp: set to: %w", err)
	}
	m.Subject(msg.Subject)
	m.SetBodyString(gomail.TypeTextHTML, msg.HTMLBody)

	if err := s.client.DialAndSendWithContext(ctx, m); err != nil {
		return fmt.Errorf("smtp: send: %w", err)
	}
	return nil
}
