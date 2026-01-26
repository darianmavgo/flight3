package flight

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"
)

// Flight now has has pocketbase as a core dependency.
// Configuration data, user data, and critical reference and testing data
//
//	is stored in pocketbase collections that should be populated if empty on first run.
func EnsureCollections(app core.App) error {
	if err := EnsureRcloneRemotes(app); err != nil {
		return err
	}
	if err := EnsureMksqliteConfigs(app); err != nil {
		return err
	}
	return EnsureDataPipelines(app)
}

func EnsureRcloneRemotes(app core.App) error {
	name := "rclone_remotes"
	existing, err := app.FindCollectionByNameOrId(name)
	if err == nil && existing != nil {
		return nil
	}

	collection := core.NewBaseCollection(name)
	collection.Fields.Add(&core.TextField{Name: "name", Required: true})
	collection.Fields.Add(&core.TextField{Name: "type", Required: true}) // e.g. s3, drive
	collection.Fields.Add(&core.JSONField{Name: "config"})               // e.g. {"access_key_id": "...", ...}

	return app.Save(collection)
}

func EnsureMksqliteConfigs(app core.App) error {
	name := "mksqlite_configs"
	existing, err := app.FindCollectionByNameOrId(name)
	if err == nil && existing != nil {
		return nil
	}

	collection := core.NewBaseCollection(name)
	collection.Fields.Add(&core.TextField{Name: "name", Required: true})
	collection.Fields.Add(&core.TextField{Name: "driver"}) // e.g. csv, json
	collection.Fields.Add(&core.JSONField{Name: "args"})   // e.g. {"delimiter": ","}

	return app.Save(collection)
}

func EnsureDataPipelines(app core.App) error {
	name := "data_pipelines"
	existing, err := app.FindCollectionByNameOrId(name)
	if err == nil && existing != nil {
		return nil
	}

	rcloneRemotes, err := app.FindCollectionByNameOrId("rclone_remotes")
	if err != nil {
		return fmt.Errorf("failed to find rclone_remotes: %w", err)
	}

	mksqliteConfigs, err := app.FindCollectionByNameOrId("mksqlite_configs")
	if err != nil {
		return fmt.Errorf("failed to find mksqlite_configs: %w", err)
	}

	collection := core.NewBaseCollection(name)
	collection.Fields.Add(&core.TextField{Name: "name", Required: true})

	// Relation to rclone_remotes
	collection.Fields.Add(&core.RelationField{
		Name:          "rclone_remote",
		CollectionId:  rcloneRemotes.Id,
		CascadeDelete: false,
		MaxSelect:     1,
	})

	collection.Fields.Add(&core.TextField{Name: "rclone_path", Required: true})

	// Relation to mksqlite_configs
	collection.Fields.Add(&core.RelationField{
		Name:          "mksqlite_config",
		CollectionId:  mksqliteConfigs.Id,
		CascadeDelete: false,
		MaxSelect:     1,
	})

	collection.Fields.Add(&core.NumberField{Name: "cache_ttl"}) // in minutes

	return app.Save(collection)
}

func EnsurePipelineCache(app core.App, cacheKey string, remote *core.Record, remotePath string, ttl float64) (string, error) {
	// Look for cache file
	cacheDir := filepath.Join(app.DataDir(), "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, cacheKey+".db"), nil
}
