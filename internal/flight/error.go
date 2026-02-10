package flight

import (
	"fmt"

	"github.com/darianmavgo/banquet"
)

// BanquetError wraps an error with additional context for banquet operations.
type BanquetError struct {
	Err        error
	Message    string
	Status     int
	Banquet    *banquet.Banquet
	Context    string
	Suggestion string
}

func (e *BanquetError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *BanquetError) Unwrap() error {
	return e.Err
}

// NewBanquetError creates a new BanquetError.
// The arguments match the usage in banquethandler.go:
// NewBanquetError(err, "Invalid banquet URL format", 400, nil, "", "")
// NewBanquetError(err, "Remote '%s' not found", 404, b, "", "")
func NewBanquetError(err error, message string, status int, b *banquet.Banquet, context, suggestion string) *BanquetError {
	return &BanquetError{
		Err:        err,
		Message:    message,
		Status:     status,
		Banquet:    b,
		Context:    context,
		Suggestion: suggestion,
	}
}
