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

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Create normalized schema tables
	createTableSQL := `
	-- Main commands table
	CREATE TABLE IF NOT EXISTS commands (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		directory TEXT NOT NULL,
		full_command TEXT NOT NULL  -- Keep for display purposes
	);
	
	-- Normalized words table to store unique words only once
	CREATE TABLE IF NOT EXISTS words (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		word TEXT NOT NULL UNIQUE
	);
	
	-- Junction table to link commands to words with position information
	CREATE TABLE IF NOT EXISTS command_word_positions (
		command_id INTEGER NOT NULL,
		word_id INTEGER NOT NULL,
		position INTEGER NOT NULL,  -- Position of word in command (0-based)
		PRIMARY KEY (command_id, word_id, position),
		FOREIGN KEY (command_id) REFERENCES commands(id) ON DELETE CASCADE,
		FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
	);
	
	CREATE INDEX IF NOT EXISTS idx_timestamp ON commands(timestamp);
	CREATE INDEX IF NOT EXISTS idx_directory ON commands(directory);
	CREATE INDEX IF NOT EXISTS idx_full_command ON commands(full_command);
	CREATE INDEX IF NOT EXISTS idx_words_word ON words(word);
	CREATE INDEX IF NOT EXISTS idx_command_word_positions_command_id ON command_word_positions(command_id);
	CREATE INDEX IF NOT EXISTS idx_command_word_positions_word_id ON command_word_positions(word_id);
	CREATE INDEX IF NOT EXISTS idx_command_word_positions_position ON command_word_positions(position);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return db, nil
}
