package devlog

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Logger provides structured logging with precise timestamps
type Logger struct {
	Source string // e.g., "server", "pocketbase", "rclone", "flight3"
	output *os.File
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Source    string                 `json:"source"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// New creates a new logger with the specified source
func New(source string) *Logger {
	return &Logger{
		Source: source,
		output: os.Stdout,
	}
}

// NewWithFile creates a logger that writes to a file
func NewWithFile(source, logPath string) (*Logger, error) {
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &Logger{
		Source: source,
		output: file,
	}, nil
}

// Close closes the log file if one was opened
func (l *Logger) Close() error {
	if l.output != os.Stdout && l.output != os.Stderr {
		return l.output.Close()
	}
	return nil
}

// log writes a log entry
func (l *Logger) log(level, message string, ctx map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000000Z"),
		Level:     level,
		Source:    l.Source,
		Message:   message,
		Context:   ctx,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}

	fmt.Fprintln(l.output, string(jsonData))
}

// Info logs an informational message
func (l *Logger) Info(message string, ctx map[string]interface{}) {
	l.log("INFO", message, ctx)
}

// Debug logs a debug message
func (l *Logger) Debug(message string, ctx map[string]interface{}) {
	l.log("DEBUG", message, ctx)
}

// Error logs an error message
func (l *Logger) Error(message string, err error, ctx map[string]interface{}) {
	if ctx == nil {
		ctx = make(map[string]interface{})
	}
	if err != nil {
		ctx["error"] = err.Error()
	}
	l.log("ERROR", message, ctx)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, ctx map[string]interface{}) {
	l.log("WARN", message, ctx)
}
