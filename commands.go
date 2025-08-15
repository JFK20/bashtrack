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

	wd, err := os.Getwd()
	if err != nil {
		wd = "unknown"
	}

	// Check if command should be excluded
	if app.shouldExclude(command) {
		return // Silently skip excluded commands
	}

	// Split command into words for word-by-word storage
	words := strings.Fields(command)
	if len(words) == 0 {
		return // Skip empty commands
	}

	fmt.Printf("  Words: %s\n", strings.Join(words, " "))

	// Use a transaction to ensure atomicity
	tx, err := app.db.Begin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error beginning transaction: %v\n", err)
		return
	}
	defer tx.Rollback() // Safe to call even after commit

	// Insert main command record
	result, err := tx.Exec(
		"INSERT INTO commands (timestamp, directory, full_command) VALUES (?, ?, ?)",
		time.Now(),
		wd,
		command,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error recording command: %v\n", err)
		return
	}

	commandID, err := result.LastInsertId()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting command ID: %v\n", err)
		return
	}

	// Insert each word with its position
	for position, word := range words {
		_, err = tx.Exec(
			"INSERT INTO command_words (command_id, word_position, word) VALUES (?, ?, ?)",
			commandID,
			position,
			word,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error recording word '%s': %v\n", word, err)
			return
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		fmt.Fprintf(os.Stderr, "Error committing transaction: %v\n", err)
	}
}

func (app *App) listCommands(cmd *cobra.Command, args []string) {
	limit, _ := cmd.Flags().GetInt("limit")
	filter, _ := cmd.Flags().GetString("filter")
	directory, _ := cmd.Flags().GetString("directory")

	query := "SELECT id, timestamp, full_command, directory FROM commands WHERE 1=1"
	var queryArgs []interface{}

	if filter != "" {
		// Search in both full command and individual words
		query += " AND (full_command LIKE ? OR id IN (SELECT command_id FROM command_words WHERE word LIKE ?))"
		queryArgs = append(queryArgs, "%"+filter+"%", "%"+filter+"%")
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

		// Load individual words for this command
		c.Words, _ = app.loadCommandWords(c.ID)

		fmt.Printf("[%d] %s\n", c.ID, c.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("    Dir: %s\n", c.Directory)
		fmt.Printf("    Cmd: %s\n", c.Command)
		if len(c.Words) > 0 {
			fmt.Printf("    Words: [%s]\n", strings.Join(c.Words, "] ["))
		}
		fmt.Println()
	}
}

// Helper function to load individual words for a command
func (app *App) loadCommandWords(commandID int) ([]string, error) {
	rows, err := app.db.Query(
		"SELECT word FROM command_words WHERE command_id = ? ORDER BY word_position",
		commandID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []string
	for rows.Next() {
		var word string
		if err := rows.Scan(&word); err != nil {
			continue
		}
		words = append(words, word)
	}

	return words, nil
}

func (app *App) searchCommands(cmd *cobra.Command, args []string) {
	pattern := args[0]

	// Enhanced search that looks in both full commands and individual words
	rows, err := app.db.Query(`
		SELECT DISTINCT c.id, c.timestamp, c.full_command, c.directory 
		FROM commands c 
		LEFT JOIN command_words cw ON c.id = cw.command_id 
		WHERE c.full_command LIKE ? OR cw.word LIKE ? 
		ORDER BY c.timestamp DESC LIMIT 50`,
		"%"+pattern+"%", "%"+pattern+"%",
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

		// Load individual words for this command
		c.Words, _ = app.loadCommandWords(c.ID)

		fmt.Printf("[%d] %s\n", c.ID, c.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("    Dir: %s\n", c.Directory)
		fmt.Printf("    Cmd: %s\n", c.Command)
		if len(c.Words) > 0 {
			fmt.Printf("    Words: [%s]\n", strings.Join(c.Words, "] ["))
		}
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

	// Most used commands (using full_command instead of command)
	fmt.Println("\nMost Used Commands:")
	rows, err = app.db.Query(`
		SELECT full_command, COUNT(*) as count 
		FROM commands 
		GROUP BY full_command 
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

	// Most used individual words
	fmt.Println("\nMost Used Words:")
	rows, err = app.db.Query(`
		SELECT word, COUNT(*) as count 
		FROM command_words 
		GROUP BY word 
		ORDER BY count DESC 
		LIMIT 15
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var word string
			var count int
			rows.Scan(&word, &count)
			fmt.Printf("  %s: %d\n", word, count)
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
	fmt.Println("Method 1 (Recommended): Using fc command")
	fmt.Println("Add the following to your ~/.bashrc:")
	fmt.Println()
	fmt.Printf("# BashTrack command recording\n")
	fmt.Printf("bashtrack_record() {\n")
	fmt.Printf("    local last_cmd=$(fc -ln -1 2>/dev/null | sed 's/^[ \\t]*//')\n")
	fmt.Printf("    if [[ -n \"$last_cmd\" && \"$last_cmd\" != bashtrack* ]]; then\n")
	fmt.Printf("        %s record \"$last_cmd\" 2>/dev/null\n", execPath)
	fmt.Printf("    fi\n")
	fmt.Printf("}\n")
	fmt.Printf("export PROMPT_COMMAND=\"${PROMPT_COMMAND:+$PROMPT_COMMAND$'\\n'}bashtrack_record\"\n")
	fmt.Println()
	fmt.Println("Method 2: Using history command (fallback)")
	fmt.Println("If Method 1 doesn't work, try:")
	fmt.Println()
	fmt.Printf("# Enable immediate history recording\n")
	fmt.Printf("shopt -s histappend\n")
	fmt.Printf("export HISTCONTROL=ignoredups:erasedups\n")
	fmt.Printf("export HISTSIZE=10000\n")
	fmt.Printf("export HISTFILESIZE=20000\n")
	fmt.Printf("export PROMPT_COMMAND=\"history -a; ${PROMPT_COMMAND:+$PROMPT_COMMAND$'\\n'}%s record\"\n", execPath)
	fmt.Println()
	fmt.Println("After adding either method to ~/.bashrc, reload it with:")
	fmt.Println("  source ~/.bashrc")
	fmt.Println()
	fmt.Println("Note: The tool automatically excludes common commands and sensitive patterns.")
	fmt.Println("You can customize exclusions using 'config add-exclude' and 'config remove-exclude'.")
}
