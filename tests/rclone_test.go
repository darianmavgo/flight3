package tests

import (
	"testing"

	"github.com/rclone/rclone/fs"
)

func TestRcloneBasic(t *testing.T) {
	// Basic placeholder call
	t.Log("Testing rclone import")
	// Using a basic function from fs package to ensure import works
	_ = fs.Version
}
