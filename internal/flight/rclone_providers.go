package flight

import (
	"fmt"
	"sort"

	"github.com/rclone/rclone/fs"

	// Import all rclone backends to populate fs.Registry
	_ "github.com/rclone/rclone/backend/all"
)

// ProviderInfo represents a simplified view of an rclone backend provider
type ProviderInfo struct {
	Name        string       `json:"name"`        // e.g. "s3", "drive", "dropbox"
	Description string       `json:"description"` // e.g. "Amazon S3 Compliant Storage Providers"
	Prefix      string       `json:"prefix"`      // URL prefix
	Options     []OptionInfo `json:"options"`     // Configuration options
	Hide        bool         `json:"hide"`        // Whether to hide from UI
}

// OptionInfo represents a configuration option for a provider
type OptionInfo struct {
	Name       string          `json:"name"`       // Field name (e.g. "access_key_id")
	Type       string          `json:"type"`       // "string", "bool", "int", "password", "duration", "size"
	Help       string          `json:"help"`       // Description
	Required   bool            `json:"required"`   // Is this field mandatory?
	IsPassword bool            `json:"isPassword"` // Should be masked in UI?
	Default    interface{}     `json:"default"`    // Default value
	Examples   []OptionExample `json:"examples"`   // Example values
	Advanced   bool            `json:"advanced"`   // Show in advanced section?
	Sensitive  bool            `json:"sensitive"`  // Contains sensitive data?
}

// OptionExample represents an example value for an option
type OptionExample struct {
	Value string `json:"value"` // Example value
	Help  string `json:"help"`  // Description of this example
}

// GetAllProviders returns a list of all supported rclone backends
func GetAllProviders() []ProviderInfo {
	var providers []ProviderInfo

	for _, regInfo := range fs.Registry {
		// Skip hidden providers
		if regInfo.Hide {
			continue
		}

		provider := ProviderInfo{
			Name:        regInfo.Name,
			Description: regInfo.Description,
			Prefix:      regInfo.Prefix,
			Hide:        regInfo.Hide,
			Options:     convertOptions(regInfo.Options),
		}

		providers = append(providers, provider)
	}

	// Sort alphabetically by name for consistent UI
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Name < providers[j].Name
	})

	return providers
}

// GetProviderOptions returns the configuration schema for a specific provider
func GetProviderOptions(providerType string) (*ProviderInfo, error) {
	regInfo, err := fs.Find(providerType)
	if err != nil {
		return nil, fmt.Errorf("provider '%s' not found: %w", providerType, err)
	}

	provider := &ProviderInfo{
		Name:        regInfo.Name,
		Description: regInfo.Description,
		Prefix:      regInfo.Prefix,
		Hide:        regInfo.Hide,
		Options:     convertOptions(regInfo.Options),
	}

	return provider, nil
}

// convertOptions converts rclone's fs.Options to our OptionInfo format
func convertOptions(fsOptions fs.Options) []OptionInfo {
	var options []OptionInfo

	for _, fsOpt := range fsOptions {
		// Skip options hidden from command line (internal use only)
		if fsOpt.Hide&fs.OptionHideCommandLine != 0 {
			continue
		}

		option := OptionInfo{
			Name:       fsOpt.Name,
			Help:       fsOpt.Help,
			Required:   fsOpt.Required,
			IsPassword: fsOpt.IsPassword,
			Advanced:   fsOpt.Advanced,
			Sensitive:  fsOpt.Sensitive,
			Type:       mapOptionType(fsOpt),
			Default:    fsOpt.Default,
		}

		// Convert examples
		if fsOpt.Examples != nil {
			option.Examples = make([]OptionExample, len(fsOpt.Examples))
			for i, ex := range fsOpt.Examples {
				option.Examples[i] = OptionExample{
					Value: ex.Value,
					Help:  ex.Help,
				}
			}
		}

		options = append(options, option)
	}

	return options
}

// mapOptionType maps rclone's option types to UI-friendly type strings
func mapOptionType(fsOpt fs.Option) string {
	// Use the Type() method which returns the string representation
	typeStr := fsOpt.Type()

	switch typeStr {
	case "bool":
		return "bool"
	case "int", "int64":
		return "int"
	case "Duration":
		return "duration"
	case "SizeSuffix":
		return "size"
	case "string":
		if fsOpt.IsPassword {
			return "password"
		}
		return "string"
	case "CommaSepList", "SpaceSepList":
		return "list"
	default:
		// Default to string for unknown types
		if fsOpt.IsPassword {
			return "password"
		}
		return "string"
	}
}
