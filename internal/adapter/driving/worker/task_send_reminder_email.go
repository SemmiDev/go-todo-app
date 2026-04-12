package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/semmidev/go-todo-app/internal/port/output"
)

// templateFS holds the compiled reminder email template.
var reminderTmpl = template.Must(template.New("reminder").Parse(reminderHTMLTemplate))

// ProcessTaskSendReminderEmail is the actual worker logic that resolves the DB references, renders the template, and sends the email.
func (p *RedisTaskProcessor) ProcessTaskSendReminderEmail(ctx context.Context, task *asynq.Task) error {
	var payload output.TaskPayloadSendReminderEmail
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", asynq.SkipRetry)
	}

	t, err := p.todoRepo.GetByID(ctx, payload.TodoID)
	if err != nil {
		// If Todo was deleted since the queue was triggered, skip
		return fmt.Errorf("failed to get todo: %w", err)
	}

	// Double-check condition incase they just completed it before the worker fired
	if t.Status() == "done" {
		p.logger.Info("skipping reminder, todo already done", slog.String("todo_id", t.ID().String()))
		return nil
	}

	u, err := p.userRepo.GetByID(ctx, t.UserID())
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	html, err := p.renderTemplate(templateData{
		UserName:        u.FullName(),
		TodoTitle:       t.Title(),
		TodoDescription: t.Description(),
		Priority:        string(t.Priority()),
		DueDate:         t.DueDate().In(time.Local).Format("Mon, 02 Jan 2006 15:04"),
		AppURL:          p.appURL,
	})
	if err != nil {
		return fmt.Errorf("failed to render HTML: %w", err)
	}

	msg := output.EmailMessage{
		To:       u.Email(),
		Subject:  fmt.Sprintf("⏰ Reminder: \"%s\" is due soon!", t.Title()),
		HTMLBody: html,
	}

	if err := p.emailSender.Send(ctx, msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	p.logger.Info("successfully processed reminder task",
		slog.String("todo_id", t.ID().String()),
		slog.String("email", u.Email()),
	)
	return nil
}

// ── Template helpers ──────────────────────────────────────────────────────────

type templateData struct {
	UserName        string
	TodoTitle       string
	TodoDescription string
	Priority        string
	DueDate         string
	AppURL          string
}

func (p *RedisTaskProcessor) renderTemplate(data templateData) (string, error) {
	var buf bytes.Buffer
	if err := reminderTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const reminderHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8"/>
  <title>Todo Reminder</title>
  <style>
    body { margin: 0; padding: 0; background-color: #F0FDFA; font-family: 'Space Grotesk', system-ui, -apple-system, sans-serif; color: #0F172A; }
    .wrapper { max-width: 600px; margin: 40px auto; padding: 20px; }
    .container { background-color: #ffffff; border: 3px solid #0F172A; box-shadow: 8px 8px 0 0 #0F172A; padding: 40px; }
    .header { margin-bottom: 30px; border-bottom: 3px solid #0F172A; padding-bottom: 20px; }
    .header h1 { margin: 0; font-size: 28px; font-weight: 800; text-transform: uppercase; letter-spacing: -0.03em; }
    .header h1 span { background-color: #06B6D4; color: #ffffff; padding: 0 8px; border: 2px solid #0F172A; display: inline-block; transform: rotate(-1deg); }
    .greeting { font-size: 18px; font-weight: 700; margin-bottom: 24px; }
    .todo-card { background-color: #ffffff; border: 3px solid #0F172A; box-shadow: 5px 5px 0 0 #0F172A; padding: 24px; margin-bottom: 30px; }
    .todo-title { font-size: 20px; font-weight: 800; margin: 0 0 8px 0; color: #0F172A; }
    .todo-desc { font-size: 16px; color: #475569; margin: 0 0 16px 0; font-weight: 500; }
    .badge { display: inline-block; padding: 4px 12px; font-size: 12px; font-weight: 800; text-transform: uppercase; border: 2px solid #0F172A; box-shadow: 2px 2px 0 0 #0F172A; margin-right: 8px; }
    .badge-low { background-color: #ECFDF5; color: #065F46; }
    .badge-medium { background-color: #FFFBEB; color: #92400E; }
    .badge-high { background-color: #FFF1F2; color: #9F1239; }
    .due-date { margin-top: 16px; font-size: 14px; font-weight: 700; color: #F43F5E; }
    .cta { text-align: center; margin-top: 40px; }
    .btn { display: inline-block; background-color: #06B6D4; color: #ffffff; text-decoration: none; padding: 16px 32px; font-weight: 800; text-transform: uppercase; border: 3px solid #0F172A; box-shadow: 5px 5px 0 0 #0F172A; font-size: 16px; }
    .footer { margin-top: 40px; text-align: center; font-size: 12px; color: #475569; font-weight: 600; }
  </style>
</head>
<body>
<div class="wrapper">
  <div class="container">
    <div class="header">
      <h1><span>⏰ REMINDER</span></h1>
    </div>
    <p class="greeting">Hi {{.UserName}},</p>
    <p style="margin-bottom: 24px; font-weight: 600;">Don't let this task slip away! It's due soon:</p>
    
    <div class="todo-card">
      <h2 class="todo-title">{{.TodoTitle}}</h2>
      {{if .TodoDescription}}<p class="todo-desc">{{.TodoDescription}}</p>{{end}}
      <div style="margin-top: 12px;">
        <span class="badge badge-{{.Priority}}">{{.Priority}}</span>
      </div>
      <div class="due-date">📅 DUE: {{.DueDate}}</div>
    </div>

    <div class="cta">
      <a href="{{.AppURL}}" class="btn">VIEW MY TODOS →</a>
    </div>

    <div class="footer">
      You're receiving this because you set a reminder for this task.<br/>
      © 2026 GO TODO APP — STAY SHARP 🚀
    </div>
  </div>
</div>
</body>
</html>`
