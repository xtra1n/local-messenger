package main

import "database/sql"

func initDB(db *sql.DB) error {
	if _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS messages (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            chat INTEGER NOT NULL,
            text TEXT NOT NULL,
            by TEXT NOT NULL,
            at DATETIME NOT NULL
        );
    `); err != nil {
		return err
	}

	if _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT NOT NULL UNIQUE,
            password_hash TEXT NOT NULL,
            created_at DATETIME NOT NULL
        );
    `); err != nil {
		return err
	}

	return nil
}
