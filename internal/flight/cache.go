package flight

import (
	"strings"

	"github.com/darianmavgo/banquet"
)

// Generate a cache key based on the banquet request.
// the auth alias "b.User" might matter for disambiguation.
// Deliberately not including scheme since file could be pulled via s3 or https in some situations.
// If dataset path is the same
func GenCacheKey(b *banquet.Banquet) string {
	return strings.Join([]string{b.User.String(), b.Hostname(), b.DataSetPath}, "-")
}
