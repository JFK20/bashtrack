package main

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// migrateDatabase handles migration from old schema to new normalized schema
func migrateDatabase(db *sql.DB) error {
	// Check if old command_words table exists
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='command_words'").Scan(&tableName)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check for old schema: %w", err)
	}

	// If old table doesn't exist, no migration needed
	if err == sql.ErrNoRows {
		return nil
	}

	// Start migration transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin migration transaction: %w", err)
	}
	defer tx.Rollback()

	// Create new tables if they don't exist
	createNewTablesSQL := `
	CREATE TABLE IF NOT EXISTS words (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		word TEXT NOT NULL UNIQUE
	);
	
	CREATE TABLE IF NOT EXISTS command_word_positions (
		command_id INTEGER NOT NULL,
		word_id INTEGER NOT NULL,
		position INTEGER NOT NULL,
		PRIMARY KEY (command_id, word_id, position),
		FOREIGN KEY (command_id) REFERENCES commands(id) ON DELETE CASCADE,
		FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
	);
	`
	if _, err := tx.Exec(createNewTablesSQL); err != nil {
		return fmt.Errorf("failed to create new tables: %w", err)
	}

	// Migrate data from old command_words to new normalized schema
	migrateDataSQL := `
	INSERT OR IGNORE INTO words (word)
	SELECT DISTINCT word FROM command_words;
	
	INSERT INTO command_word_positions (command_id, word_id, position)
	SELECT cw.command_id, w.id, cw.word_position
	FROM command_words cw
	JOIN words w ON w.word = cw.word;
	`
	if _, err := tx.Exec(migrateDataSQL); err != nil {
		return fmt.Errorf("failed to migrate data: %w", err)
	}

	// Drop old table
	if _, err := tx.Exec("DROP TABLE command_words"); err != nil {
		return fmt.Errorf("failed to drop old table: %w", err)
	}

	// Commit migration
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

func initDatabase(dbPath string) (*sql.DB, error) {
	// Add connection parameters to prevent database locking
	connectionString := fmt.Sprintf("%s?cache=shared&mode=rwc&_journal_mode=WAL&_timeout=5000", dbPath)
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	// Check if migration is needed
	if err := migrateDatabase(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
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
