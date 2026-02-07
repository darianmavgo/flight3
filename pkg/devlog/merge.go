package devlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// MergeLogs merges multiple log files chronologically and outputs to specified path
// Supports both JSON structured logs and plaintext logs with timestamps
func MergeLogs(logFiles []string, outputPath string) error {
	var entries []LogEntry

	// Read all log files
	for _, logFile := range logFiles {
		fileEntries, err := readLogFile(logFile)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", logFile, err)
		}
		entries = append(entries, fileEntries...)
	}

	// Sort by timestamp
	sort.Slice(entries, func(i, j int) bool {
		ti, _ := time.Parse("2006-01-02T15:04:05.000000Z", entries[i].Timestamp)
		tj, _ := time.Parse("2006-01-02T15:04:05.000000Z", entries[j].Timestamp)
		return ti.Before(tj)
	})

	// Write to output
	var output *os.File
	var err error
	if outputPath == "-" {
		output = os.Stdout
	} else {
		output, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer output.Close()
	}

	// Write merged logs
	for _, entry := range entries {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		fmt.Fprintln(output, string(jsonData))
	}

	return nil
}

// readLogFile reads a log file and returns entries
func readLogFile(path string) ([]LogEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry LogEntry
		// Try to parse as JSON
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			entries = append(entries, entry)
		} else {
			// If not JSON, treat as plaintext with current timestamp
			entries = append(entries, LogEntry{
				Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000000Z"),
				Level:     "INFO",
				Source:    "unknown",
				Message:   line,
			})
		}
	}

	return entries, scanner.Err()
}
