package flight

import (
	"fmt"

	"github.com/darianmavgo/banquet"
)

// BanquetError wraps an error with additional context
type BanquetError struct {
	Err       error
	Message   string
	Status    int
	Banquet   *banquet.Banquet
	Something string // Unused but present in signature
	CachePath string
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

// NewBanquetError creates a new BanquetError
func NewBanquetError(err error, message string, status int, b *banquet.Banquet, something, cachePath string) error {
	return &BanquetError{
		Err:       err,
		Message:   message,
		Status:    status,
		Banquet:   b,
		Something: something,
		CachePath: cachePath,
	}
}
