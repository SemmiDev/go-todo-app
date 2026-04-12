package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/semmidev/go-todo-app/internal/port/output"
)

// welcomeTmpl holds the compiled welcome email template.
var welcomeTmpl = template.Must(template.New("welcome").Parse(welcomeHTMLTemplate))

// ProcessTaskSendWelcomeEmail decodes the payload and sends a welcome email to the new user.
func (p *RedisTaskProcessor) ProcessTaskSendWelcomeEmail(ctx context.Context, task *asynq.Task) error {
	var payload output.TaskPayloadSendWelcomeEmail
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", asynq.SkipRetry)
	}

	u, err := p.userRepo.GetByID(ctx, payload.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	html, err := p.renderWelcomeTemplate(u.FullName())
	if err != nil {
		return fmt.Errorf("failed to render HTML: %w", err)
	}

	msg := output.EmailMessage{
		To:       u.Email(),
		Subject:  fmt.Sprintf("Welcome to Todo App, %s! 🚀", u.FullName()),
		HTMLBody: html,
	}

	if err := p.emailSender.Send(ctx, msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	p.logger.Info("successfully processed welcome email task",
		slog.String("user_id", u.ID().String()),
		slog.String("email", u.Email()),
	)
	return nil
}

func (p *RedisTaskProcessor) renderWelcomeTemplate(userName string) (string, error) {
	var buf bytes.Buffer
	data := struct {
		UserName string
		AppURL   string
	}{
		UserName: userName,
		AppURL:   p.appURL,
	}
	if err := welcomeTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const welcomeHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8"/>
  <title>Welcome to Todo App</title>
  <style>
    body{margin:0;padding:0;background:#f4f6f9;font-family:'Segoe UI',Arial,sans-serif}
    .wrapper{max-width:560px;margin:40px auto;background:#fff;border-radius:12px;box-shadow:0 4px 24px rgba(0,0,0,.08);overflow:hidden}
    .header{background:linear-gradient(135deg,#6366f1 0%,#8b5cf6 100%);padding:32px 40px;color:#fff;text-align:center}
    .header h1{margin:0;font-size:24px;font-weight:700}
    .body{padding:36px 40px;color:#1e1b4b;line-height:1.6}
    .cta{text-align:center;margin:32px 0}
    .cta a{display:inline-block;background:#6366f1;color:#fff;text-decoration:none;padding:12px 28px;border-radius:8px;font-weight:600;font-size:14px}
    .footer{padding:20px 40px;text-align:center;font-size:12px;color:#94a3b8;border-top:1px solid #f1f5f9}
  </style>
</head>
<body>
<div class="wrapper">
  <div class="header">
    <h1>Welcome aboard! 🚀</h1>
  </div>
  <div class="body">
    <p>Hi {{.UserName}},</p>
    <p>We're thrilled to have you here! Todo App is designed to help you stay organized and crush your goals with ease.</p>
    <p>Get started by creating your first task and setting up reminders so you never miss a deadline.</p>
    <div class="cta"><a href="{{.AppURL}}">Get Started Now →</a></div>
    <p>If you have any questions, feel free to reply to this email.</p>
    <p>Happy productivity!<br/>The Todo App Team</p>
  </div>
  <div class="footer">
    © Todo App — make it happen 🚀
  </div>
</div>
</body>
</html>`
