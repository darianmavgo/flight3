package flight

import (
	"context"
	"fmt"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
)

// TestRemoteConfig attempts to connect to a remote and list its root
func TestRemoteConfig(remoteType string, config map[string]interface{}) error {
	// Create configmap from provided config
	m := configmap.Simple{}
	for k, v := range config {
		m[k] = fmt.Sprintf("%v", v)
	}

	// Find the provider
	ri, err := fs.Find(remoteType)
	if err != nil {
		return fmt.Errorf("provider '%s' not found: %w", remoteType, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt to create filesystem instance
	f, err := ri.NewFs(ctx, "", "", m)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	// Try to list root directory
	entries, err := f.List(ctx, "")
	if err != nil {
		return fmt.Errorf("list operation failed: %w", err)
	}

	// Success!
	return fmt.Errorf("connection successful! Found %d items in root", len(entries))
}
