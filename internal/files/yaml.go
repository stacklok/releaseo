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
// The currentVersion is used to find embedded versions within larger values (e.g., image tags).
func UpdateYAMLFile(cfg VersionFileConfig, currentVersion, newVersion string) error {
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

	var valueAtPath string
	if err := path.Read(bytes.NewReader(data), &valueAtPath); err != nil {
		return fmt.Errorf("path %s not found in %s: %w", cfg.Path, cfg.File, err)
	}

	// Build the old and new version strings with prefix
	oldVersionStr := cfg.Prefix + currentVersion
	newVersionStr := cfg.Prefix + newVersion

	// Determine what to replace: either the embedded version or the entire value
	var oldValue, newValue string
	if strings.Contains(valueAtPath, oldVersionStr) {
		// Embedded version found - replace just the version portion
		oldValue = valueAtPath
		newValue = strings.Replace(valueAtPath, oldVersionStr, newVersionStr, 1)
	} else if embeddedVersion := findEmbeddedVersion(valueAtPath, cfg.Prefix); embeddedVersion != "" {
		// Value contains an embedded version, but it doesn't match currentVersion
		// This indicates a version mismatch that should be fixed before releasing
		return fmt.Errorf("version mismatch in %s at path %s: "+
			"expected to find %q but found %q in value %q. "+
			"This usually means the file was not updated in a previous release. "+
			"Please manually update the version in this file to %q before running releaseo",
			cfg.File, cfg.Path, oldVersionStr, embeddedVersion, valueAtPath, oldVersionStr)
	} else {
		// No embedded version - replace the entire value (original behavior)
		oldValue = valueAtPath
		newValue = newVersionStr
	}

	// Extract the key name from the path for targeted replacement
	key := extractKeyFromPath(cfg.Path)

	// Perform surgical replacement - find and replace only the value for this specific key
	newData, err := surgicalReplace(data, key, oldValue, newValue)
	if err != nil {
		return fmt.Errorf("replacing value at path %s: %w", cfg.Path, err)
	}

	// Write the file back
	if err := os.WriteFile(cfg.File, newData, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", cfg.File, err)
	}

	return nil
}

// extractKeyFromPath extracts the final key name from a dot-notation path.
// Examples:
//
//	"version" -> "version"
//	"metadata.version" -> "version"
//	"spec.containers[0].image" -> "image"
//	"operator.image" -> "image"
func extractKeyFromPath(path string) string {
	// Remove any array indices from the path
	re := regexp.MustCompile(`\[\d+\]`)
	cleanPath := re.ReplaceAllString(path, "")

	// Get the last component after splitting by dots
	parts := strings.Split(cleanPath, ".")
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}

// replacementRule defines a pattern-replacement pair for surgical YAML value replacement.
// Each rule targets a specific quote style or value format in YAML files.
type replacementRule struct {
	// name describes what this rule handles (for debugging/documentation)
	name string
	// pattern returns the regex pattern to match for the given key and old value.
	// The key is included to ensure we only match the specific YAML key we're updating.
	pattern func(key, oldValue string) string
	// replacement returns the replacement string for the given key and new value
	replacement func(key, newValue string) string
}

// replacementRules defines all the rules for surgical YAML value replacement.
// Rules are tried in order; the first matching rule is applied.
// Each pattern includes the key name to ensure we only replace the value for the
// specific YAML key being updated, not other keys with the same value.
var replacementRules = []replacementRule{
	{
		// Handles double-quoted values: key: "value"
		name: "double-quoted",
		pattern: func(key, oldValue string) string {
			return fmt.Sprintf(`(%s:\s*)"(%s)"`, regexp.QuoteMeta(key), regexp.QuoteMeta(oldValue))
		},
		replacement: func(key, newValue string) string {
			// Use ${1} syntax to avoid ambiguity when newValue starts with a digit
			return fmt.Sprintf(`${1}"%s"`, newValue)
		},
	},
	{
		// Handles single-quoted values: key: 'value'
		name: "single-quoted",
		pattern: func(key, oldValue string) string {
			return fmt.Sprintf(`(%s:\s*)'(%s)'`, regexp.QuoteMeta(key), regexp.QuoteMeta(oldValue))
		},
		replacement: func(key, newValue string) string {
			return fmt.Sprintf(`${1}'%s'`, newValue)
		},
	},
	{
		// Handles unquoted values at end of line: key: value\n
		name: "unquoted-eol",
		pattern: func(key, oldValue string) string {
			return fmt.Sprintf(`(%s:\s*)(%s)(\s*)$`, regexp.QuoteMeta(key), regexp.QuoteMeta(oldValue))
		},
		replacement: func(key, newValue string) string {
			return fmt.Sprintf(`${1}%s${3}`, newValue)
		},
	},
	{
		// Handles unquoted values followed by inline comment: key: value # comment
		name: "unquoted-with-comment",
		pattern: func(key, oldValue string) string {
			return fmt.Sprintf(`(%s:\s*)(%s)(\s*#)`, regexp.QuoteMeta(key), regexp.QuoteMeta(oldValue))
		},
		replacement: func(key, newValue string) string {
			return fmt.Sprintf(`${1}%s${3}`, newValue)
		},
	},
}

// surgicalReplace performs a targeted replacement of a YAML value while preserving
// the original formatting (quotes, whitespace, etc.). The key parameter ensures
// we only replace the value for the specific YAML key being updated.
func surgicalReplace(data []byte, key, oldValue, newValue string) ([]byte, error) {
	content := string(data)

	// Try each replacement rule in order; use the first one that matches
	for _, rule := range replacementRules {
		pattern := rule.pattern(key, oldValue)
		re := regexp.MustCompile(`(?m)` + pattern)
		if re.MatchString(content) {
			result := re.ReplaceAllString(content, rule.replacement(key, newValue))
			return []byte(result), nil
		}
	}

	// Fallback: key-aware simple string replacement if no pattern matched
	// Look for "key: oldValue" or "key:oldValue" patterns
	keyPattern := regexp.MustCompile(fmt.Sprintf(`(%s:\s*)%s`, regexp.QuoteMeta(key), regexp.QuoteMeta(oldValue)))
	if keyPattern.MatchString(content) {
		result := keyPattern.ReplaceAllString(content, fmt.Sprintf(`${1}%s`, newValue))
		return []byte(result), nil
	}

	return nil, fmt.Errorf("could not find value %q for key %q to replace", oldValue, key)
}

// findEmbeddedVersion looks for a version pattern in the value and returns it if found.
// It detects patterns like ":v1.2.3", ":1.2.3", or prefix followed by semver at end of string.
// Returns empty string if no embedded version is detected.
func findEmbeddedVersion(value, prefix string) string {
	// Pattern to match versions: optional prefix + semver (major.minor.patch with optional prerelease)
	// Looks for versions after ":" (common in image tags) or at end of string
	patterns := []string{
		// Image tag style: repo:v1.2.3 or repo:1.2.3
		`:` + regexp.QuoteMeta(prefix) + `(\d+\.\d+\.\d+(?:-[a-zA-Z0-9.]+)?)`,
		// Version at end of string with prefix
		regexp.QuoteMeta(prefix) + `(\d+\.\d+\.\d+(?:-[a-zA-Z0-9.]+)?)$`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(value); matches != nil {
			// Return the full match including prefix (reconstruct it)
			if strings.HasPrefix(matches[0], ":") {
				return matches[0][1:] // Remove leading ":"
			}
			return matches[0]
		}
	}

	return ""
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
