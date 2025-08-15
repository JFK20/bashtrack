package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func (app *App) shouldExclude(command string) bool {
	for _, pattern := range app.config.ExcludePatterns {
		matched, err := regexp.MatchString(pattern, command)
		if err != nil {
			continue // Skip invalid patterns
		}
		if matched {
			return true
		}
	}
	return false
}

func (app *App) recordCommand(cmd *cobra.Command, args []string) {
	command := strings.Join(args, " ")

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		wd = "unknown"
	}

	// Check if command should be excluded
	if app.shouldExclude(command) {
		return // Silently skip excluded commands
	}

	// Insert command into database
	_, err = app.db.Exec(
		"INSERT INTO commands (timestamp, command, directory) VALUES (?, ?, ?)",
		time.Now(),
		command,
		wd,
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error recording command: %v\n", err)
	}
}

func (app *App) listCommands(cmd *cobra.Command, args []string) {
	limit, _ := cmd.Flags().GetInt("limit")
	filter, _ := cmd.Flags().GetString("filter")
	directory, _ := cmd.Flags().GetString("directory")

	query := "SELECT id, timestamp, command, directory FROM commands WHERE 1=1"
	var queryArgs []interface{}

	if filter != "" {
		query += " AND command LIKE ?"
		queryArgs = append(queryArgs, "%"+filter+"%")
	}

	if directory != "" {
		query += " AND directory LIKE ?"
		queryArgs = append(queryArgs, "%"+directory+"%")
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	queryArgs = append(queryArgs, limit)

	rows, err := app.db.Query(query, queryArgs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying commands: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Printf("Recent Commands (limit: %d)\n", limit)
	fmt.Println(strings.Repeat("-", 80))

	for rows.Next() {
		var c Command
		err := rows.Scan(&c.ID, &c.Timestamp, &c.Command, &c.Directory)
		if err != nil {
			continue
		}

		fmt.Printf("[%d] %s\n", c.ID, c.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("    Dir: %s\n", c.Directory)
		fmt.Printf("    Cmd: %s\n", c.Command)
		fmt.Println()
	}
}

func (app *App) searchCommands(cmd *cobra.Command, args []string) {
	pattern := args[0]

	rows, err := app.db.Query(
		"SELECT id, timestamp, command, directory FROM commands WHERE command LIKE ? ORDER BY timestamp DESC LIMIT 50",
		"%"+pattern+"%",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching commands: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Printf("Commands matching '%s':\n", pattern)
	fmt.Println(strings.Repeat("-", 80))

	count := 0
	for rows.Next() {
		var c Command
		err := rows.Scan(&c.ID, &c.Timestamp, &c.Command, &c.Directory)
		if err != nil {
			continue
		}

		fmt.Printf("[%d] %s\n", c.ID, c.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("    Dir: %s\n", c.Directory)
		fmt.Printf("    Cmd: %s\n", c.Command)
		fmt.Println()
		count++
	}

	if count == 0 {
		fmt.Println("No commands found matching the pattern.")
	}
}

func (app *App) showStats(cmd *cobra.Command, args []string) {
	var totalCommands int
	err := app.db.QueryRow("SELECT COUNT(*) FROM commands").Scan(&totalCommands)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting total commands: %v\n", err)
		return
	}

	var oldestDateStr, newestDateStr sql.NullString
	err = app.db.QueryRow("SELECT MIN(timestamp), MAX(timestamp) FROM commands").Scan(&oldestDateStr, &newestDateStr)
	if err != nil && err != sql.ErrNoRows {
		fmt.Fprintf(os.Stderr, "Error getting date range: %v\n", err)
		return
	}

	fmt.Println("Command Tracking Statistics")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Total commands: %d\n\n", totalCommands)

	if oldestDateStr.Valid && newestDateStr.Valid {
		// Parse the timestamp strings to time.Time
		oldestDate, err1 := time.Parse("2006-01-02 15:04:05.999999999-07:00", oldestDateStr.String)
		newestDate, err2 := time.Parse("2006-01-02 15:04:05.999999999-07:00", newestDateStr.String)

		if err1 == nil && err2 == nil {
			fmt.Printf("Date range: %s to %s\n",
				oldestDate.Format("2006-01-02"),
				newestDate.Format("2006-01-02"))

			days := int(newestDate.Sub(oldestDate).Hours() / 24)
			if days > 0 {
				fmt.Printf("Average per day: %.1f\n", float64(totalCommands)/float64(days))
			}
		} else {
			fmt.Println("Error parsing date strings.")
		}
	}

	// Top directories
	fmt.Println("\nTop Directories:")
	rows, err := app.db.Query(`
		SELECT directory, COUNT(*) as count 
		FROM commands 
		GROUP BY directory 
		ORDER BY count DESC 
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dir string
			var count int
			rows.Scan(&dir, &count)
			fmt.Printf("  %s: %d\n", dir, count)
		}
	}

	// Most used commands
	fmt.Println("\nMost Used Commands:")
	rows, err = app.db.Query(`
		SELECT command, COUNT(*) as count 
		FROM commands 
		GROUP BY command 
		ORDER BY count DESC 
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var command string
			var count int
			rows.Scan(&command, &count)
			// Truncate long commands
			if len(command) > 50 {
				command = command[:50] + "..."
			}
			fmt.Printf("  %s: %d\n", command, count)
		}
	}
}

func (app *App) showConfig(cmd *cobra.Command, args []string) {
	fmt.Println("Current Configuration:")
	fmt.Println(strings.Repeat("=", 30))
	fmt.Printf("Database: %s\n", app.config.DatabasePath)
	fmt.Println("\nExclude Patterns:")
	for i, pattern := range app.config.ExcludePatterns {
		fmt.Printf("  %d. %s\n", i+1, pattern)
	}
}

func (app *App) addExcludePattern(cmd *cobra.Command, args []string) {
	pattern := args[0]

	// Check if pattern already exists
	for _, existing := range app.config.ExcludePatterns {
		if existing == pattern {
			fmt.Printf("Pattern '%s' already exists\n", pattern)
			return
		}
	}

	app.config.ExcludePatterns = append(app.config.ExcludePatterns, pattern)

	configDir, _ := getConfigDir()
	configPath := filepath.Join(configDir, configFile)

	_, err := saveConfig(configPath, app.config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		return
	}

	fmt.Printf("Added exclude pattern: %s\n", pattern)
}

func (app *App) removeExcludePattern(cmd *cobra.Command, args []string) {
	pattern := args[0]

	for i, existing := range app.config.ExcludePatterns {
		if existing == pattern {
			app.config.ExcludePatterns = append(app.config.ExcludePatterns[:i], app.config.ExcludePatterns[i+1:]...)

			configDir, _ := getConfigDir()
			configPath := filepath.Join(configDir, configFile)

			_, err := saveConfig(configPath, app.config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
				return
			}

			fmt.Printf("Removed exclude pattern: %s\n", pattern)
			return
		}
	}

	fmt.Printf("Pattern '%s' not found\n", pattern)
}

func (app *App) cleanupCommands(cmd *cobra.Command, args []string) {
	days, _ := cmd.Flags().GetInt("days")
	cutoff := time.Now().AddDate(0, 0, -days)

	result, err := app.db.Exec("DELETE FROM commands WHERE timestamp < ?", cutoff)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error cleaning up commands: %v\n", err)
		return
	}

	affected, _ := result.RowsAffected()
	fmt.Printf("Removed %d commands older than %d days\n", affected, days)
}

func (app *App) showSetupInstructions(cmd *cobra.Command, args []string) {
	execPath, err := os.Executable()
	if err != nil {
		execPath = appName
	}

	fmt.Println("Bash Command Tracker Setup Instructions")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()
	fmt.Println("To start tracking your bash commands, add the following line to your ~/.bashrc file:")
	fmt.Println()
	fmt.Printf(`export PROMPT_COMMAND="${PROMPT_COMMAND:+$PROMPT_COMMAND$'\n'}%s record \"$(history 1 | sed 's/^[ ]*[0-9]*[ ]*//')\""`, execPath)
	fmt.Println()
	fmt.Println()
	fmt.Println("Then reload your bash configuration:")
	fmt.Println("  source ~/.bashrc")
	fmt.Println()
	fmt.Println("Note: The tool automatically excludes common commands and sensitive patterns.")
	fmt.Println("You can customize exclusions using 'config add-exclude' and 'config remove-exclude'.")
}
