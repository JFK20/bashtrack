package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

const (
	appName    = "bashtrack"
	configFile = "config.json"
	dbFile     = "commands.db"
)

type Config struct {
	ExcludePatterns []string `json:"exclude_patterns"`
	DatabasePath    string   `json:"database_path"`
}

type Command struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Command   string    `json:"command"`
	Directory string    `json:"directory"`
	Words     []string  `json:"words"`
}

type App struct {
	db     *sql.DB
	config *Config
}

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	rootCmd := &cobra.Command{
		Use:   appName,
		Short: "Track and manage bash command history",
		Long:  "A CLI tool to track all bash commands in an SQLite database with filtering and search capabilities.",
	}

	// Add command to record a new command
	recordCmd := &cobra.Command{
		Use:   "record [command]",
		Short: "Record a command to the database",
		Args:  cobra.MinimumNArgs(1),
		Run:   app.recordCommand,
	}

	// Add command to list recent commands
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List recent commands",
		Run:   app.listCommands,
	}
	listCmd.Flags().IntP("limit", "l", 20, "Number of commands to show")
	listCmd.Flags().StringP("filter", "f", "", "Filter commands by pattern")
	listCmd.Flags().StringP("directory", "d", "", "Filter by directory")

	// Add command to search commands
	searchCmd := &cobra.Command{
		Use:   "search [pattern]",
		Short: "Search commands by pattern",
		Args:  cobra.ExactArgs(1),
		Run:   app.searchCommands,
	}

	// Add command to show statistics
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show command statistics",
		Run:   app.showStats,
	}

	// Add command to manage configuration
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	configShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Run:   app.showConfig,
	}

	configAddExcludeCmd := &cobra.Command{
		Use:   "add-exclude [pattern]",
		Short: "Add an exclude pattern",
		Args:  cobra.ExactArgs(1),
		Run:   app.addExcludePattern,
	}

	configRemoveExcludeCmd := &cobra.Command{
		Use:   "remove-exclude [pattern]",
		Short: "Remove an exclude pattern",
		Args:  cobra.ExactArgs(1),
		Run:   app.removeExcludePattern,
	}

	// Add setup command
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Show setup instructions",
		Run:   app.showSetupInstructions,
	}

	// Add cleanup command
	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove old commands",
		Run:   app.cleanupCommands,
	}
	cleanupCmd.Flags().IntP("days", "d", 90, "Remove commands older than this many days")

	configCmd.AddCommand(configShowCmd, configAddExcludeCmd, configRemoveExcludeCmd)
	rootCmd.AddCommand(recordCmd, listCmd, searchCmd, statsCmd, configCmd, setupCmd, cleanupCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func NewApp() (*App, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load or create config
	config, err := loadConfig(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize database
	db, err := initDatabase(config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &App{
		db:     db,
		config: config,
	}, nil
}

func (app *App) Close() {
	if app.db != nil {
		app.db.Close()
	}
}
