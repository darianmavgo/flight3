package flight

import (
	"fmt"

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
	if err := EnsureAppSettings(app); err != nil {
		return err
	}
	if err := EnsureBanquetLinks(app); err != nil {
		return err
	}
	return EnsureRecentFiles(app)
}

func EnsureAppSettings(app core.App) error {
	name := "app_settings"
	existing, err := app.FindCollectionByNameOrId(name)
	if err == nil && existing != nil {
		return nil
	}

	collection := core.NewBaseCollection(name)
	collection.Fields.Add(&core.TextField{Name: "value", Required: true})
	// Add unique constraint on key using an index? PocketBase handles unique constraints via indexes,
	// but programmatic API for indexes is specific. For now, simple fields are fine.

	// Make public
	collection.ListRule = nil
	collection.ViewRule = nil
	collection.CreateRule = nil
	collection.UpdateRule = nil
	collection.DeleteRule = nil
	// For public read/write, rules should be likely empty string "" or nil (nil means admin only usually in PB, empty string means public!)
	// Wait, nil usually means Admin Only in PocketBase. "" (empty string) means Public.
	// Let's check PB docs or existing code.
	// Actually, looking at other PB examples: nil is Admin only. "" is Public.
	public := ""
	collection.ListRule = &public
	collection.ViewRule = &public
	collection.CreateRule = &public
	collection.UpdateRule = &public
	collection.DeleteRule = &public

	return app.Save(collection)
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

	// Make public
	public := ""
	collection.ListRule = &public
	collection.ViewRule = &public
	collection.CreateRule = &public
	collection.UpdateRule = &public
	collection.DeleteRule = &public

	return app.Save(collection)
}

func EnsureRecentFiles(app core.App) error {
	name := "recent_files"
	existing, err := app.FindCollectionByNameOrId(name)
	if err == nil && existing != nil {
		return nil
	}

	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		return fmt.Errorf("failed to find superusers collection: %w", err)
	}

	collection := core.NewBaseCollection(name)

	// Relation to user
	collection.Fields.Add(&core.RelationField{
		Name:          "user",
		CollectionId:  superusers.Id,
		CascadeDelete: true, // Delete recent files when user is deleted
		MaxSelect:     1,
		Required:      true,
	})

	collection.Fields.Add(&core.TextField{Name: "path", Required: true})
	collection.Fields.Add(&core.TextField{Name: "name", Required: true})
	collection.Fields.Add(&core.DateField{Name: "last_opened", Required: true})
	collection.Fields.Add(&core.BoolField{Name: "was_converted"})
	collection.Fields.Add(&core.TextField{Name: "original_format"})

	// Make public
	public := ""
	collection.ListRule = &public
	collection.ViewRule = &public
	collection.CreateRule = &public
	collection.UpdateRule = &public
	collection.DeleteRule = &public

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

	// Make public
	public := ""
	collection.ListRule = &public
	collection.ViewRule = &public
	collection.CreateRule = &public
	collection.UpdateRule = &public
	collection.DeleteRule = &public

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

	// Make public
	public := ""
	collection.ListRule = &public
	collection.ViewRule = &public
	collection.CreateRule = &public
	collection.UpdateRule = &public
	collection.DeleteRule = &public

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

	// Make public
	public := ""
	collection.ListRule = &public
	collection.ViewRule = &public
	collection.CreateRule = &public
	collection.UpdateRule = &public
	collection.DeleteRule = &public

	return app.Save(collection)
}

func EnsureSuperUser(app core.App, email, password string) error {
	superuser, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, email)
	if err != nil {
		collection, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
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
