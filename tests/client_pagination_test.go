package tests

import (
	"testing"

	"github.com/darianmavgo/banquet"
	"github.com/stretchr/testify/assert"
)

func TestClientSidePaginationViaAnchor(t *testing.T) {
	// This test validates the DESIGN, not implementation details of the server.

	// URL pattern: data.csv;tb0#page=5
	// Flight3 should:
	// 1. Ignore #anchor (browsers usually don't send this to the server anyway)
	// 2. Serve full SQLite file if requested via /sqliter/file/

	url := "data.csv;tb0#page=5"
	// ParseNested should handle this since it uses url.Parse
	b, err := banquet.ParseNested(url)
	assert.NoError(t, err)

	// Banquet should strip anchor from DataSetPath
	assert.Equal(t, "data.csv", b.DataSetPath)
	assert.Equal(t, "tb0", b.Table)

	// The anchor is in b.Fragment if parsed as a full URL,
	// but banquet.ParseNested often treats the input as a path.
}
