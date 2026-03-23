package domain

import (
	"context"
	"time"
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type UserStore interface {
	CreateUser(ctx context.Context, username, password string) error
	GetUserByUsername(ctx context.Context, username string) (User, error)
}
