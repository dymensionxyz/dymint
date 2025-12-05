package da

import (
	"fmt"
	"os"
	"strings"
)

// LoadSecret loads a secret value with the following priority:
// 1. Environment variable (if envName is not empty and variable is set)
// 2. File (if filePath is not empty and file exists)
// 3. Direct value (if directValue is not empty)
// Returns an error if none of the above provide a value.
func LoadSecret(envName, filePath, directValue, secretType string) (string, error) {
	// Try environment variable first
	if envName != "" {
		if value := os.Getenv(envName); value != "" {
			return value, nil
		}
	}

	// Try file second
	if filePath != "" {
		data, err := os.ReadFile(filePath) //nolint:gosec // filePath is from config, intentional file read for secrets
		if err != nil {
			// Only return error if file was explicitly specified
			return "", fmt.Errorf("read %s file %s: %w", secretType, filePath, err)
		}
		value := strings.TrimSpace(string(data))
		if value != "" {
			return value, nil
		}
	}

	// Fall back to direct value
	if directValue != "" {
		return directValue, nil
	}

	// Build helpful error message
	var options []string
	if envName != "" {
		options = append(options, fmt.Sprintf("environment variable %s", envName))
	}
	if filePath != "" {
		options = append(options, fmt.Sprintf("file %s", filePath))
	}
	options = append(options, fmt.Sprintf("%s field in config", secretType))

	return "", fmt.Errorf("%s not found: set one of: %s", secretType, strings.Join(options, ", "))
}
