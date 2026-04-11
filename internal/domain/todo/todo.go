package todo

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ─── Enums ────────────────────────────────────────────────────────────────────

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

func (s Status) IsValid() bool {
	return s == StatusPending || s == StatusInProgress || s == StatusDone
}

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

func (p Priority) IsValid() bool {
	return p == PriorityLow || p == PriorityMedium || p == PriorityHigh
}

// ─── Tag value object ─────────────────────────────────────────────────────────

type Tag struct {
	id        uuid.UUID
	userID    uuid.UUID
	name      string
	color     string
	createdAt time.Time
	updatedAt time.Time
}

func NewTag(userID uuid.UUID, name, color string) (*Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("tag name is required")
	}
	if len(name) > 50 {
		return nil, errors.New("tag name must be at most 50 characters")
	}
	if color == "" {
		color = "#6366f1"
	}
	now := time.Now().UTC()
	return &Tag{
		id: uuid.Must(uuid.NewV7()), userID: userID,
		name: name, color: color,
		createdAt: now, updatedAt: now,
	}, nil
}

func ReconstituteTag(id, userID uuid.UUID, name, color string, createdAt, updatedAt time.Time) *Tag {
	return &Tag{id: id, userID: userID, name: name, color: color, createdAt: createdAt, updatedAt: updatedAt}
}

func (t *Tag) ID() uuid.UUID        { return t.id }
func (t *Tag) UserID() uuid.UUID    { return t.userID }
func (t *Tag) Name() string         { return t.name }
func (t *Tag) Color() string        { return t.color }
func (t *Tag) CreatedAt() time.Time { return t.createdAt }
func (t *Tag) UpdatedAt() time.Time { return t.updatedAt }

func (t *Tag) Update(name, color string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("tag name is required")
	}
	if len(name) > 50 {
		return errors.New("tag name must be at most 50 characters")
	}
	t.name = name
	if color != "" {
		t.color = color
	}
	t.updatedAt = time.Now().UTC()
	return nil
}

// ─── Todo aggregate root ──────────────────────────────────────────────────────

type Todo struct {
	id           uuid.UUID
	userID       uuid.UUID
	title        string
	description  string
	status       Status
	priority     Priority
	dueDate      *time.Time
	createdAt    time.Time
	updatedAt    time.Time
	tags         []*Tag
	reminderSent bool
}

func New(userID uuid.UUID, title, description string, priority Priority, dueDate *time.Time) (*Todo, error) {
	if userID == uuid.Nil {
		return nil, errors.New("user_id is required")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("title is required")
	}
	if len(title) > 200 {
		return nil, errors.New("title must be at most 200 characters")
	}
	if !priority.IsValid() {
		priority = PriorityMedium
	}
	now := time.Now().UTC()
	return &Todo{
		id: uuid.Must(uuid.NewV7()), userID: userID,
		title: title, description: strings.TrimSpace(description),
		status: StatusPending, priority: priority,
		dueDate: dueDate, createdAt: now, updatedAt: now,
		tags: []*Tag{},
	}, nil
}

func Reconstitute(
	id, userID uuid.UUID,
	title, description string,
	status Status, priority Priority,
	dueDate *time.Time,
	createdAt, updatedAt time.Time,
	tags []*Tag,
	reminderSent bool,
) *Todo {
	if tags == nil {
		tags = []*Tag{}
	}
	return &Todo{
		id: id, userID: userID,
		title: title, description: description,
		status: status, priority: priority,
		dueDate: dueDate, createdAt: createdAt, updatedAt: updatedAt,
		tags: tags, reminderSent: reminderSent,
	}
}

func (t *Todo) ID() uuid.UUID                { return t.id }
func (t *Todo) UserID() uuid.UUID            { return t.userID }
func (t *Todo) Title() string                { return t.title }
func (t *Todo) Description() string          { return t.description }
func (t *Todo) Status() Status               { return t.status }
func (t *Todo) Priority() Priority           { return t.priority }
func (t *Todo) DueDate() *time.Time          { return t.dueDate }
func (t *Todo) CreatedAt() time.Time         { return t.createdAt }
func (t *Todo) UpdatedAt() time.Time         { return t.updatedAt }
func (t *Todo) Tags() []*Tag                 { return t.tags }
func (t *Todo) ReminderSent() bool           { return t.reminderSent }
func (t *Todo) IsOwnedBy(uid uuid.UUID) bool { return t.userID == uid }

// MarkReminderSent records that a reminder email has been sent for this todo.
func (t *Todo) MarkReminderSent() {
	t.reminderSent = true
}

func (t *Todo) Update(title, description string, status Status, priority Priority, dueDate *time.Time) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("title is required")
	}
	if len(title) > 200 {
		return errors.New("title must be at most 200 characters")
	}
	if !status.IsValid() {
		return errors.New("invalid status")
	}
	if !priority.IsValid() {
		priority = PriorityMedium
	}
	t.title = title
	t.description = strings.TrimSpace(description)
	t.status = status
	t.priority = priority
	t.dueDate = dueDate
	t.updatedAt = time.Now().UTC()
	return nil
}

func (t *Todo) SetTags(tags []*Tag) {
	t.tags = tags
	t.updatedAt = time.Now().UTC()
}

func (t *Todo) AddTag(tag *Tag) {
	for _, existing := range t.tags {
		if existing.ID() == tag.ID() {
			return
		}
	}
	t.tags = append(t.tags, tag)
	t.updatedAt = time.Now().UTC()
}

func (t *Todo) RemoveTag(tagID uuid.UUID) {
	filtered := t.tags[:0]
	for _, tag := range t.tags {
		if tag.ID() != tagID {
			filtered = append(filtered, tag)
		}
	}
	t.tags = filtered
	t.updatedAt = time.Now().UTC()
}
