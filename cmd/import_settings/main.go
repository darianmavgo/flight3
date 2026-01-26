package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	// 1. Locate Config Files
	// We assume they are in sibling directories as per user hint
	workDir, _ := os.Getwd()
	baseDir := filepath.Dir(workDir) // ../ from flight3

	// List of potential repos to check
	repos := []string{"sqliter", "banquet", "mksqlite", "TableTypeMaster"}

	configs := make(map[string]map[string]string)

	for _, repo := range repos {
		configPath := filepath.Join(baseDir, repo, "config.hcl")
		if _, err := os.Stat(configPath); err == nil {
			log.Printf("Found config: %s", configPath)
			settings, err := parseHCL(configPath)
			if err != nil {
				log.Printf("Error parsing %s: %v", configPath, err)
				continue
			}
			configs[repo] = settings
		}
	}

	// 2. Import into PocketBase
	dataDir := filepath.Join(workDir, "user_settings", "pb_data")
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: dataDir,
	})

	// Bind operation to OnServe to ensure DB is ready
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		collection, err := app.FindCollectionByNameOrId("app_settings")
		if err != nil {
			// Create if missing?
			// In ensuringCollections we usually do this.
			// Let's assume it exists or fail.
			return fmt.Errorf("app_settings collection not found: %v", err)
		}

		for repo, settings := range configs {
			log.Printf("Importing settings for %s...", repo)
			for k, v := range settings {
				// Check if exists
				// We use 'key' field as unique identifier?
				// app_settings schema from ensureCollections check (not visible here but implied)
				// usually: key, value.

				// Construct a unique key if needed, or overwrite global keys?
				// `sqliter` has `port`=8080. `flight3` might not want to overwrite its port.
				// But user asked to import.
				// Maybe prefix? "sqliter.port"? Or just "port"?
				// "import those key value pairs" -> implies direct mapping.
				// But typically app_settings has 'key', 'value'.

				// Let's check if record with 'key' exists.
				record, err := app.FindFirstRecordByData("app_settings", "key", k)
				if err != nil || record == nil {
					record = core.NewRecord(collection)
					record.Set("key", k)
				}

				record.Set("value", v)
				// Maybe add a 'source' field if it exists?
				// Only if schema supports it. Safer to just set k/v.

				if err := app.Save(record); err != nil {
					log.Printf("Failed to save %s=%s: %v", k, v, err)
				} else {
					log.Printf("Set %s = %s", k, v)
				}
			}
		}
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// Simple Regex HCL Parser
// Matches: key = "value" or key = true
func parseHCL(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	results := make(map[string]string)
	scanner := bufio.NewScanner(f)

	// Regex for: key = value
	// value can be quoted string, boolean, or number
	re := regexp.MustCompile(`^\s*([a-zA-Z0-9_]+)\s*=\s*(.*)\s*$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			key := matches[1]
			val := matches[2]

			// Unquote if string
			if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
				val = strings.Trim(val, "\"")
			}
			// Keep booleans/numbers as string representation

			results[key] = val
		}
	}

	return results, scanner.Err()
}
