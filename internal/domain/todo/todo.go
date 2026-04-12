// Package todo defines the Todo and Tag domain models.
// It includes logic for managing tasks, their lifecycle, and categorization.
package todo

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ─── Enums ────────────────────────────────────────────────────────────────────

// Status represents the current lifecycle state of a Todo.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

// IsValid checks if the status matches one of the predefined constants.
func (s Status) IsValid() bool {
	return s == StatusPending || s == StatusInProgress || s == StatusDone
}

// Priority represents the importance level of a Todo.
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// IsValid checks if the priority matches one of the predefined constants.
func (p Priority) IsValid() bool {
	return p == PriorityLow || p == PriorityMedium || p == PriorityHigh
}

// ─── Tag value object ─────────────────────────────────────────────────────────

// Tag is a labels that can be attached to Todos for categorization.
type Tag struct {
	id        uuid.UUID
	userID    uuid.UUID
	name      string
	color     string
	createdAt time.Time
	updatedAt time.Time
}

// NewTag creates a new Tag with validation and safe defaults for colors.
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

// ReconstituteTag creates a Tag from existing data (e.g., from a database).
func ReconstituteTag(id, userID uuid.UUID, name, color string, createdAt, updatedAt time.Time) *Tag {
	return &Tag{id: id, userID: userID, name: name, color: color, createdAt: createdAt, updatedAt: updatedAt}
}

// ID returns the unique identifier for the tag.
func (t *Tag) ID() uuid.UUID        { return t.id }

// UserID returns the owner's identifier.
func (t *Tag) UserID() uuid.UUID    { return t.userID }

// Name returns the tag's display name.
func (t *Tag) Name() string         { return t.name }

// Color returns the tag's hexadecimal color code.
func (t *Tag) Color() string        { return t.color }

// CreatedAt returns the timestamp when the tag was created.
func (t *Tag) CreatedAt() time.Time { return t.createdAt }

// UpdatedAt returns the timestamp when the tag was last modified.
func (t *Tag) UpdatedAt() time.Time { return t.updatedAt }

// Update modifies the tag's name and color with validation.
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

// Todo is the central aggregate root representing a task.
type Todo struct {
	id                 uuid.UUID
	userID             uuid.UUID
	title              string
	description        string
	status             Status
	priority           Priority
	dueDate            *time.Time
	createdAt          time.Time
	updatedAt          time.Time
	tags               []*Tag
	reminders          []string // e.g., ["1h", "15m"]
	triggeredReminders []string // e.g., ["1h"]
}

// New creates a new Todo with default status (Pending) and validated fields.
func New(userID uuid.UUID, title, description string, priority Priority, dueDate *time.Time, reminders []string) (*Todo, error) {
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
		tags: []*Tag{}, reminders: reminders, triggeredReminders: []string{},
	}, nil
}

// Reconstitute creates a Todo from existing data, ensuring internal collections are initialized.
func Reconstitute(
	id, userID uuid.UUID,
	title, description string,
	status Status, priority Priority,
	dueDate *time.Time,
	createdAt, updatedAt time.Time,
	tags []*Tag,
	reminders, triggeredReminders []string,
) *Todo {
	if tags == nil {
		tags = []*Tag{}
	}
	if reminders == nil {
		reminders = []string{}
	}
	if triggeredReminders == nil {
		triggeredReminders = []string{}
	}
	return &Todo{
		id: id, userID: userID,
		title: title, description: description,
		status: status, priority: priority,
		dueDate: dueDate, createdAt: createdAt, updatedAt: updatedAt,
		tags: tags, reminders: reminders, triggeredReminders: triggeredReminders,
	}
}

// ID returns the unique identifier for the todo.
func (t *Todo) ID() uuid.UUID                { return t.id }

// UserID returns the owner's identifier.
func (t *Todo) UserID() uuid.UUID            { return t.userID }

// Title returns the todo's short headline.
func (t *Todo) Title() string                { return t.title }

// Description returns the detailed content of the todo.
func (t *Todo) Description() string          { return t.description }

// Status returns the current state (Pending, InProgress, Done).
func (t *Todo) Status() Status               { return t.status }

// Priority returns the importance level (Low, Medium, High).
func (t *Todo) Priority() Priority           { return t.priority }

// DueDate returns the optional deadline for the task.
func (t *Todo) DueDate() *time.Time          { return t.dueDate }

// CreatedAt returns the timestamp when the todo was created.
func (t *Todo) CreatedAt() time.Time         { return t.createdAt }

// UpdatedAt returns the timestamp when the todo was last modified.
func (t *Todo) UpdatedAt() time.Time         { return t.updatedAt }

// Tags returns the list of associated tags.
func (t *Todo) Tags() []*Tag { return t.tags }

// Reminders returns the list of configured reminder offsets (e.g., "1h").
func (t *Todo) Reminders() []string { return t.reminders }

// TriggeredReminders returns the list of offsets that have already been sent.
func (t *Todo) TriggeredReminders() []string { return t.triggeredReminders }

// IsOwnedBy returns true if the provided user ID matches the owner.
func (t *Todo) IsOwnedBy(uid uuid.UUID) bool { return t.userID == uid }

// MarkReminderTriggered records that a specific reminder offset has been sent.
func (t *Todo) MarkReminderTriggered(offset string) {
	for _, tr := range t.triggeredReminders {
		if tr == offset {
			return
		}
	}
	t.triggeredReminders = append(t.triggeredReminders, offset)
}

// GetDueReminders returns a list of reminder offsets that are now due to be sent
// but have not been triggered yet.
func (t *Todo) GetDueReminders(now time.Time) []string {
	if t.status == StatusDone || t.dueDate == nil {
		return nil
	}

	triggeredMap := make(map[string]bool)
	for _, offset := range t.triggeredReminders {
		triggeredMap[offset] = true
	}

	var due []string
	for _, offset := range t.reminders {
		if triggeredMap[offset] {
			continue
		}

		d, err := time.ParseDuration(offset)
		if err != nil {
			continue
		}

		// (due_date - offset) <= now
		if t.dueDate.Add(-d).Before(now) || t.dueDate.Add(-d).Equal(now) {
			due = append(due, offset)
		}
	}
	return due
}

// TransitionStatus moves the todo to a new status based on allowed transitions.
// Allowed:
// - pending -> in_progress
// - in_progress -> pending
// - in_progress -> done
// - done -> in_progress
// - done -> pending
// Not Allowed:
// - pending -> done (directly)
func (t *Todo) TransitionStatus(newStatus Status) error {
	if !newStatus.IsValid() {
		return errors.New("invalid status")
	}

	if t.status == newStatus {
		return nil
	}

	// State machine logic
	switch t.status {
	case StatusPending:
		if newStatus == StatusDone {
			return errors.New("cannot transition directly from pending to done")
		}
	case StatusInProgress:
		// All transitions allowed from in_progress
	case StatusDone:
		// All transitions allowed from done
	}

	t.status = newStatus
	t.updatedAt = time.Now().UTC()
	return nil
}

// Update modifies core fields of the Todo with validation.
func (t *Todo) Update(title, description string, status Status, priority Priority, dueDate *time.Time, reminders []string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("title is required")
	}
	if len(title) > 200 {
		return errors.New("title must be at most 200 characters")
	}
	if err := t.TransitionStatus(status); err != nil {
		return err
	}
	t.title = title
	t.description = strings.TrimSpace(description)
	t.priority = priority
	t.dueDate = dueDate
	t.reminders = reminders
	t.updatedAt = time.Now().UTC()
	return nil
}

// SetTags replaces the entire tag collection for the Todo.
func (t *Todo) SetTags(tags []*Tag) {
	t.tags = tags
	t.updatedAt = time.Now().UTC()
}

// AddTag appends a new tag to the Todo if it doesn't already exist in the list.
func (t *Todo) AddTag(tag *Tag) {
	for _, existing := range t.tags {
		if existing.ID() == tag.ID() {
			return
		}
	}
	t.tags = append(t.tags, tag)
	t.updatedAt = time.Now().UTC()
}

// RemoveTag detaches a tag from the Todo by its ID.
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
