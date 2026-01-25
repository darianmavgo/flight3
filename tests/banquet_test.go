package tests

import (
	"testing"

	"github.com/darianmavgo/banquet"
)

func TestBanquetBasic(t *testing.T) {
	// Basic usage to verify import
	t.Log("Testing banquet import")
	_, err := banquet.ParseBanquet("http://example.com")
	if err != nil {
		t.Logf("ParseBanquet returned error (expected with dummy URL): %v", err)
	}
}
