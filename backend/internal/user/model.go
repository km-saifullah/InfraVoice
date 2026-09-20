package user

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"

	StatusActive  = "active"
	StatusBlocked = "blocked"
)

type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string        `bson:"name" json:"name"`
	Email        string        `bson:"email" json:"email"`
	PasswordHash string        `bson:"password_hash" json:"-"`
	Role         string        `bson:"role" json:"role"`
	Status       string        `bson:"status" json:"status"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at" json:"updated_at"`
}

func (u *User) Normalize() {
	u.Name = strings.TrimSpace(u.Name)
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
}

func (u *User) IsActive() bool {
	return u.Status == StatusActive
}
