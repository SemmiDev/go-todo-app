package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	id        uuid.UUID
	email     string
	fullName  string
	createdAt time.Time
	updatedAt time.Time
}

func New(email, fullName string) *User {
	now := time.Now().UTC()
	return &User{
		id: uuid.Must(uuid.NewV7()), email: email,
		fullName: fullName, createdAt: now, updatedAt: now,
	}
}

func Reconstitute(id uuid.UUID, email, fullName string, createdAt, updatedAt time.Time) *User {
	return &User{id: id, email: email, fullName: fullName, createdAt: createdAt, updatedAt: updatedAt}
}

func (u *User) ID() uuid.UUID        { return u.id }
func (u *User) Email() string        { return u.email }
func (u *User) FullName() string     { return u.fullName }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) UpdatedAt() time.Time { return u.updatedAt }
