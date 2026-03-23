package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/xtra1n/local-messenger/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type sqliteUserStore struct {
	db *sql.DB
}

func NewSQLiteUserStore(db *sql.DB) domain.UserStore {
	return &sqliteUserStore{db: db}
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (s *sqliteUserStore) CreateUser(ctx context.Context, username, password string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, created_at)
         VALUES (?, ?, ?)`,
		username, hash, time.Now(),
	)

	return err
}

func (s *sqliteUserStore) GetUserByUsername(ctx context.Context, username string) (domain.User, error) {
	var u domain.User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, created_at
         FROM users
         WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)

	return u, err
}
