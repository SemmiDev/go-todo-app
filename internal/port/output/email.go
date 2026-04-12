// Package output defines the outbound ports (interfaces) of the application.
// These interfaces are implemented by the driven adapters (infrastructure) and called by the application layer.
package output

import "context"

// EmailMessage represents an outbound email with its recipients and content.
type EmailMessage struct {
	// To is the recipient's email address.
	To string
	// Subject is the email header's topic.
	Subject string
	// HTMLBody is the rich-text content of the email.
	HTMLBody string
}

// EmailSender is the driven port for sending outbound communications.
type EmailSender interface {
	// Send dispatches an email message to its recipient.
	Send(ctx context.Context, msg EmailMessage) error
}
