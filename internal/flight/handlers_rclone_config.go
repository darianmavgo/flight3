package flight

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pocketbase/pocketbase/core"
)

// HandleRcloneConfigUI serves the main HTML page
func HandleRcloneConfigUI(e *core.RequestEvent) error {
	// Read template file
	templatePath := filepath.Join("templates", "rclone_config.html")
	htmlContent, err := os.ReadFile(templatePath)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load template: " + err.Error(),
		})
	}

	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = e.Response.Write(htmlContent)
	return err
}

// HandleListProviders returns JSON array of all rclone providers
func HandleListProviders(e *core.RequestEvent) error {
	providers := GetAllProviders()
	return e.JSON(http.StatusOK, map[string]interface{}{
		"providers": providers,
	})
}

// HandleGetProviderSchema returns configuration schema for a specific provider
func HandleGetProviderSchema(e *core.RequestEvent) error {
	providerType := e.Request.PathValue("type")
	if providerType == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "provider type is required",
		})
	}

	provider, err := GetProviderOptions(providerType)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}

	return e.JSON(http.StatusOK, provider)
}

// HandleListRemotes queries rclone_remotes collection
func HandleListRemotes(e *core.RequestEvent) error {
	records, err := e.App.FindRecordsByFilter(
		"rclone_remotes",
		"", // no filter, get all
		"", // remove sort to avoid "invalid sort field" if created is missing
		100,
		0,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to fetch remotes: %v", err),
		})
	}

	// Convert to JSON-friendly format
	remotes := make([]map[string]interface{}, len(records))
	for i, record := range records {
		remotes[i] = record.PublicExport()
	}

	return e.JSON(http.StatusOK, map[string]interface{}{
		"remotes": remotes,
	})
}

// RemoteCreateRequest represents the payload for creating a remote
type RemoteCreateRequest struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	VFSSettings map[string]interface{} `json:"vfs_settings"`
	Enabled     bool                   `json:"enabled"`
	Description string                 `json:"description"`
}

// HandleCreateRemote validates and creates new remote record
func HandleCreateRemote(e *core.RequestEvent) error {
	var req RemoteCreateRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid JSON: %v", err),
		})
	}

	// Validate remote name (alphanumeric + underscores, no spaces)
	if !isValidRemoteName(req.Name) {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "remote name must be alphanumeric with underscores only (no spaces)",
		})
	}

	// Check for duplicate name
	existing, _ := e.App.FindFirstRecordByFilter("rclone_remotes", "name = {:name}", map[string]interface{}{
		"name": req.Name,
	})
	if existing != nil {
		return e.JSON(http.StatusConflict, map[string]string{
			"error": fmt.Sprintf("remote with name '%s' already exists", req.Name),
		})
	}

	// Validate provider type exists
	if _, err := GetProviderOptions(req.Type); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid provider type: %v", err),
		})
	}

	// Get collection
	collection, err := e.App.FindCollectionByNameOrId("rclone_remotes")
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "rclone_remotes collection not found",
		})
	}

	// Create record
	record := core.NewRecord(collection)
	record.Set("name", req.Name)
	record.Set("type", req.Type)
	record.Set("enabled", req.Enabled)
	record.Set("description", req.Description)

	// Set config as JSON
	if req.Config != nil {
		record.Set("config", req.Config)
	}

	// Set VFS settings as JSON
	if req.VFSSettings != nil {
		record.Set("vfs_settings", req.VFSSettings)
	}

	if err := e.App.Save(record); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to save remote: %v", err),
		})
	}

	return e.JSON(http.StatusCreated, record.PublicExport())
}

// HandleUpdateRemote patches existing remote record
func HandleUpdateRemote(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if id == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "remote ID is required",
		})
	}

	var req RemoteCreateRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid JSON: %v", err),
		})
	}

	// Find existing record
	record, err := e.App.FindRecordById("rclone_remotes", id)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "remote not found",
		})
	}

	// Validate new name if changing
	if req.Name != "" && req.Name != record.GetString("name") {
		if !isValidRemoteName(req.Name) {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "remote name must be alphanumeric with underscores only (no spaces)",
			})
		}

		// Check for duplicate
		existing, _ := e.App.FindFirstRecordByFilter("rclone_remotes", "name = {:name} && id != {:id}", map[string]interface{}{
			"name": req.Name,
			"id":   id,
		})
		if existing != nil {
			return e.JSON(http.StatusConflict, map[string]string{
				"error": fmt.Sprintf("remote with name '%s' already exists", req.Name),
			})
		}

		record.Set("name", req.Name)
	}

	// Update fields
	if req.Type != "" {
		// Validate provider type
		if _, err := GetProviderOptions(req.Type); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("invalid provider type: %v", err),
			})
		}
		record.Set("type", req.Type)
	}

	if req.Config != nil {
		record.Set("config", req.Config)
	}

	if req.VFSSettings != nil {
		record.Set("vfs_settings", req.VFSSettings)
	}

	record.Set("enabled", req.Enabled)
	record.Set("description", req.Description)

	if err := e.App.Save(record); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to update remote: %v", err),
		})
	}

	return e.JSON(http.StatusOK, record.PublicExport())
}

// HandleDeleteRemote removes remote record
func HandleDeleteRemote(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if id == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "remote ID is required",
		})
	}

	record, err := e.App.FindRecordById("rclone_remotes", id)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "remote not found",
		})
	}

	if err := e.App.Delete(record); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to delete remote: %v", err),
		})
	}

	return e.JSON(http.StatusOK, map[string]string{
		"message": "remote deleted successfully",
	})
}

// isValidRemoteName validates remote name format
func isValidRemoteName(name string) bool {
	// Allow alphanumeric, underscores, and hyphens
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched && name != ""
}

// TestRequest represents the payload for testing a remote config
type TestRequest struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// HandleTestRemote tests a remote configuration
func HandleTestRemote(e *core.RequestEvent) error {
	var req TestRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid JSON: %v", err),
		})
	}

	if req.Type == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "provider type is required",
		})
	}

	// Test the configuration
	err := TestRemoteConfig(req.Type, req.Config)
	if err != nil {
		// Check if it's actually a success message
		if err.Error()[:21] == "connection successful" {
			return e.JSON(http.StatusOK, map[string]interface{}{
				"success": true,
				"message": err.Error(),
			})
		}
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return e.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Connection test successful",
	})
}
