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
    body{margin:0;padding:0;background:#f4f6f9;font-family:'Segoe UI',Arial,sans-serif}
    .wrapper{max-width:560px;margin:40px auto;background:#fff;border-radius:12px;box-shadow:0 4px 24px rgba(0,0,0,.08);overflow:hidden}
    .header{background:linear-gradient(135deg,#6366f1 0%,#8b5cf6 100%);padding:32px 40px;color:#fff}
    .header h1{margin:0;font-size:22px;font-weight:700}
    .header p{margin:6px 0 0;opacity:.85;font-size:14px}
    .body{padding:36px 40px}
    .card{background:#f8f9ff;border:1.5px solid #e0e3ff;border-radius:10px;padding:20px 24px;margin-bottom:24px}
    .title{font-size:18px;font-weight:600;color:#1e1b4b;margin:0 0 10px}
    .desc{color:#64748b;font-size:14px;margin:0 0 10px}
    .badge{display:inline-block;padding:3px 10px;border-radius:999px;font-size:12px;font-weight:600;text-transform:uppercase}
    .badge-low{background:#dcfce7;color:#166534}
    .badge-medium{background:#fef9c3;color:#854d0e}
    .badge-high{background:#fee2e2;color:#991b1b}
    .due{font-size:13px;color:#dc2626;font-weight:600;margin-top:12px}
    .cta{text-align:center;margin-top:8px}
    .cta a{display:inline-block;background:#6366f1;color:#fff;text-decoration:none;padding:12px 28px;border-radius:8px;font-weight:600;font-size:14px}
    .footer{padding:20px 40px;text-align:center;font-size:12px;color:#94a3b8;border-top:1px solid #f1f5f9}
  </style>
</head>
<body>
<div class="wrapper">
  <div class="header">
    <h1>⏰ Reminder: Todo Due Soon</h1>
    <p>Hi {{.UserName}}, don't let this slip through!</p>
  </div>
  <div class="body">
    <div class="card">
      <div class="title">{{.TodoTitle}}</div>
      {{if .TodoDescription}}<p class="desc">{{.TodoDescription}}</p>{{end}}
      <span class="badge badge-{{.Priority}}">{{.Priority}}</span>
      <div class="due">📅 Due: {{.DueDate}}</div>
    </div>
    <div class="cta"><a href="{{.AppURL}}">Open Todo App →</a></div>
  </div>
  <div class="footer">
    You received this because this todo is due within 24 hours.<br/>
    © Todo App — keep crushing it 🚀
  </div>
</div>
</body>
</html>`
