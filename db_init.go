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

	// Create tables for word-by-word storage
	createTableSQL := `
	-- Main commands table
	CREATE TABLE IF NOT EXISTS commands (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		directory TEXT NOT NULL,
		full_command TEXT NOT NULL  -- Keep for display purposes
	);
	
	-- Words table to store individual words
	CREATE TABLE IF NOT EXISTS command_words (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		command_id INTEGER NOT NULL,
		word_position INTEGER NOT NULL,  -- Position of word in command (0-based)
		word TEXT NOT NULL,
		FOREIGN KEY (command_id) REFERENCES commands(id) ON DELETE CASCADE
	);
	
	CREATE INDEX IF NOT EXISTS idx_timestamp ON commands(timestamp);
	CREATE INDEX IF NOT EXISTS idx_directory ON commands(directory);
	CREATE INDEX IF NOT EXISTS idx_full_command ON commands(full_command);
	CREATE INDEX IF NOT EXISTS idx_command_words_command_id ON command_words(command_id);
	CREATE INDEX IF NOT EXISTS idx_command_words_word ON command_words(word);
	CREATE INDEX IF NOT EXISTS idx_command_words_position ON command_words(word_position);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return db, nil
}
