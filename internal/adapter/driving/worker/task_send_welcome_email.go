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
    body { margin: 0; padding: 0; background-color: #F0FDFA; font-family: 'Space Grotesk', system-ui, -apple-system, sans-serif; color: #0F172A; }
    .wrapper { max-width: 600px; margin: 40px auto; padding: 20px; }
    .container { background-color: #ffffff; border: 3px solid #0F172A; box-shadow: 8px 8px 0 0 #0F172A; padding: 40px; }
    .header { margin-bottom: 30px; border-bottom: 3px solid #0F172A; padding-bottom: 20px; text-align: center; }
    .header h1 { margin: 0; font-size: 28px; font-weight: 800; text-transform: uppercase; letter-spacing: -0.03em; }
    .header h1 span { background-color: #06B6D4; color: #ffffff; padding: 0 12px; border: 2px solid #0F172A; display: inline-block; transform: rotate(1.5deg); }
    .greeting { font-size: 20px; font-weight: 800; margin-bottom: 24px; }
    .content { font-size: 16px; color: #475569; line-height: 1.6; font-weight: 600; }
    .feature-card { background-color: #CFFAFE; border: 3px solid #0F172A; box-shadow: 4px 4px 0 0 #0F172A; padding: 20px; margin: 24px 0; }
    .feature-card p { margin: 0; color: #0F172A; font-weight: 700; }
    .cta { text-align: center; margin-top: 40px; }
    .btn { display: inline-block; background-color: #06B6D4; color: #ffffff; text-decoration: none; padding: 16px 32px; font-weight: 800; text-transform: uppercase; border: 3px solid #0F172A; box-shadow: 5px 5px 0 0 #0F172A; font-size: 16px; }
    .footer { margin-top: 40px; text-align: center; font-size: 12px; color: #475569; font-weight: 600; border-top: 2px solid #0F172A; padding-top: 20px; }
  </style>
</head>
<body>
<div class="wrapper">
  <div class="container">
    <div class="header">
      <h1><span>WELCOME ABOARD!</span></h1>
    </div>
    
    <div class="content">
      <p class="greeting">HI {{.UserName}},</p>
      <p>We're thrilled to have you here! Todo App is built for people who want to get things done without the fluff.</p>
      
      <div class="feature-card">
        <p>🚀 Ready to crush your goals? Start by creating your first task and setting up a reminder.</p>
      </div>

      <p>No more missed deadlines. No more forgotten ideas. Just pure productivity.</p>
    </div>

    <div class="cta">
      <a href="{{.AppURL}}" class="btn">GET STARTED NOW →</a>
    </div>

    <div class="footer">
      © 2026 GO TODO APP — MAKE IT HAPPEN 🚀
    </div>
  </div>
</div>
</body>
</html>`
