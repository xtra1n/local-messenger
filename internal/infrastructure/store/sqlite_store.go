package store

import (
	"context"
	"database/sql"

	"github.com/xtra1n/local-messenger/internal/domain"
)

type sqliteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) domain.MessageStore {
	return &sqliteStore{db: db}
}

func (s *sqliteStore) SaveMessage(ctx context.Context, msg domain.Message) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO Messages (chat_id, by_user, text, at) VALUES (?, ?, ?, ?)`,
		msg.Chat, msg.By, msg.Text, msg.At,
	)

	return err
}

func (s *sqliteStore) GetRecentMessages(ctx context.Context, chatID int, limit int) ([]domain.Message, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT chat_id, by_user, text, at
         FROM Messages
         WHERE chat_id = ?
         ORDER BY id DESC
         LIMIT ?`,
		chatID, limit,
	)

	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var msgs []domain.Message
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(&m.Chat, &m.By, &m.Text, &m.At); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}

	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, rows.Err()
}
