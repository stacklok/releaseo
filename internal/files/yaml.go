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
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
)

// VersionFileConfig defines a YAML file and the path to update with the new version.
type VersionFileConfig struct {
	File   string `json:"file"`
	Path   string `json:"path"`
	Prefix string `json:"prefix,omitempty"`
}

// UpdateYAMLFile updates a specific path in a YAML file with a new version.
// It preserves the original file formatting including comments and structure.
func UpdateYAMLFile(cfg VersionFileConfig, version string) error {
	// Read the file content
	data, err := os.ReadFile(cfg.File)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", cfg.File, err)
	}

	// Parse the YAML file into an AST
	file, err := parser.ParseBytes(data, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing file %s: %w", cfg.File, err)
	}

	// Apply prefix if specified
	newValue := version
	if cfg.Prefix != "" {
		newValue = cfg.Prefix + version
	}

	// Convert dot notation path to YAML path format
	yamlPath := convertToYAMLPath(cfg.Path)

	// Create the path
	path, err := yaml.PathString(yamlPath)
	if err != nil {
		return fmt.Errorf("creating path %s: %w", yamlPath, err)
	}

	// Validate path exists by trying to read the current value
	var currentValue interface{}
	if err := path.Read(bytes.NewReader(data), &currentValue); err != nil {
		return fmt.Errorf("path %s not found in %s: %w", cfg.Path, cfg.File, err)
	}

	// Parse the new value as YAML
	newValueYAML, err := parser.ParseBytes([]byte(newValue), 0)
	if err != nil {
		return fmt.Errorf("parsing new value: %w", err)
	}

	// Replace the value at the path
	if err := path.ReplaceWithFile(file, newValueYAML); err != nil {
		return fmt.Errorf("replacing value at path %s: %w", cfg.Path, err)
	}

	// Write the file back
	if err := os.WriteFile(cfg.File, []byte(file.String()), 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", cfg.File, err)
	}

	return nil
}

// convertToYAMLPath converts a dot notation path to YAML path format.
// Examples:
//
//	"metadata.version" -> "$.metadata.version"
//	"containers[0].image" -> "$.containers[0].image"
//	"spec.template.spec.image.tag" -> "$.spec.template.spec.image.tag"
func convertToYAMLPath(path string) string {
	// If already starts with $, return as is
	if strings.HasPrefix(path, "$") {
		return path
	}
	return "$." + path
}
