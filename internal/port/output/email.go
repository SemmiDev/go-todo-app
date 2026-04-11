package output

import "context"

// EmailMessage is a simple value object for outbound emails.
type EmailMessage struct {
	To       string
	Subject  string
	HTMLBody string
}

// EmailSender is the driven port for sending emails.
// Implementations live in internal/adapter/driven/smtp.
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}
