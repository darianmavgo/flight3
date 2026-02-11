package flight

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Import all rclone backends for testing
	_ "github.com/rclone/rclone/backend/all"
)

func TestGetAllProviders(t *testing.T) {
	providers := GetAllProviders()

	// Should have many providers (rclone supports 40+)
	assert.Greater(t, len(providers), 10, "Should have at least 10 providers")

	// Check for common providers
	providerNames := make(map[string]bool)
	for _, p := range providers {
		providerNames[p.Name] = true
	}

	expectedProviders := []string{"s3", "drive", "dropbox", "onedrive", "box", "local"}
	for _, expected := range expectedProviders {
		assert.True(t, providerNames[expected], "Should have provider: %s", expected)
	}

	// Verify structure
	for _, p := range providers {
		assert.NotEmpty(t, p.Name, "Provider should have name")
		assert.NotEmpty(t, p.Description, "Provider should have description")
		// Options can be empty for some providers
		t.Logf("Provider: %s - %s (%d options)", p.Name, p.Description, len(p.Options))
	}
}

func TestGetProviderOptions(t *testing.T) {
	t.Run("valid provider s3", func(t *testing.T) {
		provider, err := GetProviderOptions("s3")
		require.NoError(t, err)
		require.NotNil(t, provider)

		assert.Equal(t, "s3", provider.Name)
		assert.NotEmpty(t, provider.Description)
		assert.Greater(t, len(provider.Options), 0, "S3 should have options")

		// Check for common S3 options
		optionNames := make(map[string]OptionInfo)
		for _, opt := range provider.Options {
			optionNames[opt.Name] = opt
		}

		// S3 should have provider option
		if providerOpt, ok := optionNames["provider"]; ok {
			assert.NotEmpty(t, providerOpt.Examples, "Provider option should have examples")
		}

		// Check for access credentials
		assert.Contains(t, optionNames, "access_key_id", "Should have access_key_id")
		assert.Contains(t, optionNames, "secret_access_key", "Should have secret_access_key")

		// Secret key should be marked as sensitive
		if secretOpt, ok := optionNames["secret_access_key"]; ok {
			assert.True(t, secretOpt.Sensitive, "secret_access_key should be sensitive")
			// Type could be "password" or "string" depending on rclone version
			t.Logf("secret_access_key type: %s, isPassword: %v", secretOpt.Type, secretOpt.IsPassword)
		}
	})

	t.Run("valid provider drive", func(t *testing.T) {
		provider, err := GetProviderOptions("drive")
		require.NoError(t, err)
		require.NotNil(t, provider)

		assert.Equal(t, "drive", provider.Name)
		assert.Contains(t, provider.Description, "Google", "Drive description should mention Google")
	})

	t.Run("invalid provider", func(t *testing.T) {
		provider, err := GetProviderOptions("nonexistent")
		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestOptionTypeMapping(t *testing.T) {
	// Get a provider with various option types
	provider, err := GetProviderOptions("s3")
	require.NoError(t, err)

	optionsByName := make(map[string]OptionInfo)
	for _, opt := range provider.Options {
		optionsByName[opt.Name] = opt
	}

	// Test different type mappings
	tests := []struct {
		optionName   string
		expectedType string
	}{
		{"secret_access_key", "password"},
		{"acl", "string"},
		// Add more as needed based on actual S3 options
	}

	for _, tt := range tests {
		if opt, ok := optionsByName[tt.optionName]; ok {
			assert.Equal(t, tt.expectedType, opt.Type,
				"Option %s should have type %s", tt.optionName, tt.expectedType)
		}
	}
}

func TestOptionExamples(t *testing.T) {
	provider, err := GetProviderOptions("s3")
	require.NoError(t, err)

	// Find provider option which should have examples
	for _, opt := range provider.Options {
		if opt.Name == "provider" {
			assert.NotEmpty(t, opt.Examples, "Provider option should have examples")

			// Verify example structure
			for _, ex := range opt.Examples {
				assert.NotEmpty(t, ex.Value, "Example should have value")
				// Help can be empty
			}
			return
		}
	}
}

func TestAdvancedOptions(t *testing.T) {
	provider, err := GetProviderOptions("s3")
	require.NoError(t, err)

	hasBasic := false
	hasAdvanced := false

	for _, opt := range provider.Options {
		if opt.Advanced {
			hasAdvanced = true
		} else {
			hasBasic = true
		}
	}

	assert.True(t, hasBasic, "Should have basic options")
	assert.True(t, hasAdvanced, "Should have advanced options")
}

func TestRequiredFields(t *testing.T) {
	provider, err := GetProviderOptions("s3")
	require.NoError(t, err)

	// Most providers have some required fields
	// For S3, depends on provider but typically has some required options
	hasRequired := false
	for _, opt := range provider.Options {
		if opt.Required {
			hasRequired = true
			t.Logf("Required option: %s", opt.Name)
		}
	}

	// Note: Not all providers have required fields, so we just log
	t.Logf("S3 has required fields: %v", hasRequired)
}
