// Copyright 2025 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package files

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

// VersionFileConfig defines a YAML file and the path to update with the new version.
type VersionFileConfig struct {
	File   string `json:"file"`
	Path   string `json:"path"`
	Prefix string `json:"prefix,omitempty"`
}

// UpdateYAMLFile updates a specific path in a YAML file with a new version.
// It uses surgical text replacement to preserve the original file formatting.
func UpdateYAMLFile(cfg VersionFileConfig, version string) error {
	// Read the file content
	data, err := os.ReadFile(cfg.File)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", cfg.File, err)
	}

	// Convert dot notation path to YAML path format
	yamlPath, err := convertToYAMLPath(cfg.Path)
	if err != nil {
		return fmt.Errorf("invalid path %s: %w", cfg.Path, err)
	}

	// Create the path and read current value to validate it exists
	path, err := yaml.PathString(yamlPath)
	if err != nil {
		return fmt.Errorf("creating path %s: %w", yamlPath, err)
	}

	var currentValue string
	if err := path.Read(bytes.NewReader(data), &currentValue); err != nil {
		return fmt.Errorf("path %s not found in %s: %w", cfg.Path, cfg.File, err)
	}

	// Apply prefix (empty prefix just results in version)
	newValue := cfg.Prefix + version

	// Perform surgical replacement - find and replace only the value
	newData, err := surgicalReplace(data, currentValue, newValue)
	if err != nil {
		return fmt.Errorf("replacing value at path %s: %w", cfg.Path, err)
	}

	// Write the file back
	if err := os.WriteFile(cfg.File, newData, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", cfg.File, err)
	}

	return nil
}

// surgicalReplace performs a targeted replacement of a YAML value while preserving
// the original formatting (quotes, whitespace, etc.)
func surgicalReplace(data []byte, oldValue, newValue string) ([]byte, error) {
	content := string(data)

	// Try different quote styles that might wrap the value
	patterns := []string{
		// Double quoted: key: "value"
		fmt.Sprintf(`"(%s)"`, regexp.QuoteMeta(oldValue)),
		// Single quoted: key: 'value'
		fmt.Sprintf(`'(%s)'`, regexp.QuoteMeta(oldValue)),
		// Unquoted after colon: key: value
		fmt.Sprintf(`: (%s)(\s*)$`, regexp.QuoteMeta(oldValue)),
		fmt.Sprintf(`: (%s)(\s*#)`, regexp.QuoteMeta(oldValue)),
	}

	replacements := []string{
		fmt.Sprintf(`"%s"`, newValue),
		fmt.Sprintf(`'%s'`, newValue),
		fmt.Sprintf(`: %s$2`, newValue),
		fmt.Sprintf(`: %s$2`, newValue),
	}

	for i, pattern := range patterns {
		re := regexp.MustCompile(`(?m)` + pattern)
		if re.MatchString(content) {
			result := re.ReplaceAllString(content, replacements[i])
			return []byte(result), nil
		}
	}

	// Fallback: simple string replacement if no pattern matched
	if strings.Contains(content, oldValue) {
		result := strings.Replace(content, oldValue, newValue, 1)
		return []byte(result), nil
	}

	return nil, fmt.Errorf("could not find value %q to replace", oldValue)
}

// convertToYAMLPath converts a dot notation path to YAML path format.
// Examples:
//
//	"metadata.version" -> "$.metadata.version"
//	"containers[0].image" -> "$.containers[0].image"
//	"spec.template.spec.image.tag" -> "$.spec.template.spec.image.tag"
func convertToYAMLPath(path string) (string, error) {
	// Validate path is not empty
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Reject paths starting with '.' as they would create recursive descent queries
	// which can cause unexpected behavior (e.g., ".image.tag" becomes "$..image.tag")
	if strings.HasPrefix(path, ".") {
		return "", fmt.Errorf("path cannot start with '.' (got %q) - use %q instead", path, strings.TrimPrefix(path, "."))
	}

	// If already starts with $, return as is
	if strings.HasPrefix(path, "$") {
		return path, nil
	}
	return "$." + path, nil
}
