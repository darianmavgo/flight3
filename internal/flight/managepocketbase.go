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
	if err := EnsureDataPipelines(app); err != nil {
		return err
	}
	return EnsureBanquetLinks(app)
}

func EnsureBanquetLinks(app core.App) error {
	name := "banquet_links"
	existing, err := app.FindCollectionByNameOrId(name)
	if err == nil && existing != nil {
		return nil
	}

	collection := core.NewBaseCollection(name)
	// System fields (id, created, updated) are added by NewBaseCollection by default in modern PB,
	// but let's be safe and let them be if they are there.

	collection.Fields.Add(&core.TextField{Name: "original_url"})
	collection.Fields.Add(&core.TextField{Name: "scheme"})
	collection.Fields.Add(&core.TextField{Name: "user"})
	collection.Fields.Add(&core.TextField{Name: "host"})
	collection.Fields.Add(&core.TextField{Name: "path"})
	collection.Fields.Add(&core.URLField{Name: "explore_link"})
	collection.Fields.Add(&core.TextField{Name: "datasetpath"})
	collection.Fields.Add(&core.TextField{Name: "columnset"})
	collection.Fields.Add(&core.TextField{Name: "query"})

	return app.Save(collection)
}

func EnsureRcloneRemotes(app core.App) error {
	name := "rclone_remotes"
	existing, err := app.FindCollectionByNameOrId(name)
	if err == nil && existing != nil {
		return nil
	}

	collection := core.NewBaseCollection(name)
	collection.Fields.Add(&core.TextField{Name: "name", Required: true})
	collection.Fields.Add(&core.TextField{Name: "type", Required: true})    // e.g. s3, drive
	collection.Fields.Add(&core.JSONField{Name: "config"})                  // e.g. {"access_key_id": "...", ...}
	collection.Fields.Add(&core.JSONField{Name: "vfs_settings"})            // Optional VFS tuning per remote
	collection.Fields.Add(&core.BoolField{Name: "enabled", Required: true}) // Enable/disable remote
	collection.Fields.Add(&core.TextField{Name: "description"})             // Documentation

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

func EnsureSuperUser(app core.App, email, password string) error {
	superuser, err := app.FindAuthRecordByEmail("_superusers", email)
	if err != nil {
		collection, err := app.FindCollectionByNameOrId("_superusers")
		if err != nil {
			return err
		}
		record := core.NewRecord(collection)
		record.SetEmail(email)
		record.SetPassword(password)
		if err := app.Save(record); err != nil {
			return err
		}
		fmt.Printf("Created superuser %s\n", email)
	} else {
		superuser.SetPassword(password)
		if err := app.Save(superuser); err != nil {
			return err
		}
		fmt.Printf("Ensured superuser %s password is correct\n", email)
	}
	return nil
}
