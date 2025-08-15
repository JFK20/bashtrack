package main

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func initDatabase(dbPath string) (*sql.DB, error) {
	// Add connection parameters to prevent database locking
	connectionString := fmt.Sprintf("%s?cache=shared&mode=rwc&_journal_mode=WAL&_timeout=5000", dbPath)
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	// Create table if not exists
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS commands (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		command TEXT NOT NULL,
		directory TEXT NOT NULL
	);
	
	CREATE INDEX IF NOT EXISTS idx_timestamp ON commands(timestamp);
	CREATE INDEX IF NOT EXISTS idx_command ON commands(command);
	CREATE INDEX IF NOT EXISTS idx_directory ON commands(directory);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return db, nil
}
