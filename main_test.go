package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShouldExclude(t *testing.T) {
	app := &App{
		config: &Config{
			ExcludePatterns: []string{
				"^ls$",
				"^cd ",
				"bashtrack.*",
			},
		},
	}

	tests := []struct {
		command  string
		expected bool
	}{
		{"ls", true},
		{"ls -la", false},
		{"cd /tmp", true},
		{"bashtrack record", true},
		{"git status", false},
		{"echo hello", false},
	}

	for _, test := range tests {
		result := app.shouldExclude(test.command)
		if result != test.expected {
			t.Errorf("shouldExclude(%q) = %v, expected %v", test.command, result, test.expected)
		}
	}
}

func TestConfigOperations(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Test creating a new config
	config := &Config{
		ExcludePatterns: []string{"test1", "test2"},
		DatabasePath:    filepath.Join(tempDir, "test.db"),
	}

	// Test saving config
	_, err := saveConfig(configPath, config)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Test loading config
	loadedConfig, err := loadConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(loadedConfig.ExcludePatterns) != 2 {
		t.Errorf("Expected 2 exclude patterns, got %d", len(loadedConfig.ExcludePatterns))
	}

	if loadedConfig.ExcludePatterns[0] != "test1" {
		t.Errorf("Expected first pattern to be 'test1', got %q", loadedConfig.ExcludePatterns[0])
	}
}

func TestDatabaseInitialization(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Test database initialization
	db, err := initDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Verify that tables exist
	tables := []string{"commands", "words", "command_word_positions"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to check table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("Table %s does not exist", table)
		}
	}
}

func TestCommandRecording(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := initDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	app := &App{
		db: db,
		config: &Config{
			ExcludePatterns: []string{},
			DatabasePath:    dbPath,
		},
	}

	// Test recording a command
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Record a test command
	testCommand := "git status --porcelain"
	words := []string{"git", "status", "--porcelain"}

	// Mock the recordCommand function by calling it directly
	app.recordCommand(nil, words)

	// Verify the command was recorded
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM commands WHERE full_command = ?", testCommand).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query commands: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 command recorded, got %d", count)
	}

	// Verify words were recorded
	err = db.QueryRow("SELECT COUNT(*) FROM words").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query words: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 words recorded, got %d", count)
	}

	// Verify word positions were recorded
	err = db.QueryRow("SELECT COUNT(*) FROM command_word_positions").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query word positions: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 word positions recorded, got %d", count)
	}
}

func TestCommandDeduplication(t *testing.T) {
	// Create a temporary directory and database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := initDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	app := &App{
		db: db,
		config: &Config{
			ExcludePatterns: []string{},
			DatabasePath:    dbPath,
		},
	}

	// Record the same command twice
	testCommand := []string{"echo", "hello"}
	app.recordCommand(nil, testCommand)
	time.Sleep(time.Millisecond) // Ensure different timestamp
	app.recordCommand(nil, testCommand)

	// Verify only one command was recorded (deduplication)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM commands WHERE full_command = ?", "echo hello").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query commands: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 command recorded (deduplicated), got %d", count)
	}
}
