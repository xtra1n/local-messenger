package main

import "database/sql"

func initDB(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS messages (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id  INTEGER NOT NULL,
    by_user  TEXT    NOT NULL,
    text     TEXT    NOT NULL,
    at       DATETIME NOT NULL
);
`
	_, err := db.Exec(schema)
	return err
}
