package flight

import (
	"github.com/darianmavgo/banquet"
)

// ErrorResponse matches PocketBase's standard error format
type ErrorResponse struct {
	Data    map[string]interface{} `json:"data"`
	Message string                 `json:"message"`
	Status  int                    `json:"status"`
	Debug   *DebugInfo             `json:"debug,omitempty"` // Only included if debug mode
}

// DebugInfo contains debugging information for development
type DebugInfo struct {
	Banquet string `json:"banquet,omitempty"`
	Query   string `json:"query,omitempty"`
	DBPath  string `json:"db_path,omitempty"`
	Error   string `json:"error,omitempty"` // Original error message
}

// IsDebugMode checks if debug mode is enabled via environment variables

// SendBanquetError sends a JSON error response with optional debug info

// WrapBanquetError wraps an error with banquet context for later error handling
type BanquetError struct {
	Err     error
	Message string
	Status  int
	Banquet *banquet.Banquet
	Query   string
	DBPath  string
}

func (be *BanquetError) Error() string {
	return be.Message
}

func (be *BanquetError) Unwrap() error {
	return be.Err
}

// NewBanquetError creates a new BanquetError
func NewBanquetError(err error, message string, status int, b *banquet.Banquet, query string, dbPath string) *BanquetError {
	return &BanquetError{
		Err:     err,
		Message: message,
		Status:  status,
		Banquet: b,
		Query:   query,
		DBPath:  dbPath,
	}
}

// HandleBanquetError checks if an error is a BanquetError and sends appropriate response
