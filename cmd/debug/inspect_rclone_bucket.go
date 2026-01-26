package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/darianmavgo/flight3/internal/secrets"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/rclone/rclone/backend/all"
	rcfs "github.com/rclone/rclone/fs"
)

func main() {
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// 1. Secrets Setup
	dbPath := filepath.Join(workDir, "user_settings", "user_secrets.db")
	keyPath := filepath.Join(workDir, "key")

	s, err := secrets.NewService(dbPath, keyPath)
	if err != nil {
		log.Fatalf("Secrets error: %v", err)
	}
	defer s.Close()

	// 2. Get Credentials for r2-auth
	creds, err := s.GetCredentials("r2-auth")
	if err != nil {
		log.Fatalf("GetCredentials error: %v", err)
	}

	// 3. Build Rclone Connection String
	remoteType, _ := creds["type"].(string)

	var sb strings.Builder
	sb.WriteString(":")
	sb.WriteString(remoteType)

	for k, v := range creds {
		if k == "type" {
			continue
		}
		valStr := fmt.Sprintf("%v", v)
		escaped := strings.ReplaceAll(valStr, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		sb.WriteString(fmt.Sprintf(",%s=\"%s\"", k, escaped))
	}

	connStr := sb.String()

	// 4. Connect to Root
	ctx := context.Background()
	fSys, err := rcfs.NewFs(ctx, connStr)
	if err != nil {
		log.Fatalf("NewFs error: %v", err)
	}

	fmt.Printf("Root FS Name: %v\n", fSys.Name())
	fmt.Printf("Root FS Root: %v\n", fSys.Root())

	// 5. Connect specifically to the bucket path and check Features
	fmt.Println("\n--- Bucket FS Analysis ---")
	bucketFs, err := rcfs.NewFs(ctx, connStr+":test-mksqlite")
	if err != nil {
		log.Printf("Bucket FS error: %v", err)
	} else {
		fmt.Printf("Bucket FS Name: %v\n", bucketFs.Name())
		fmt.Printf("Bucket FS Root: %v\n", bucketFs.Root())
		// Check features
		features := bucketFs.Features()
		fmt.Printf("BucketBased: %v\n", features.BucketBased)
	}
}
