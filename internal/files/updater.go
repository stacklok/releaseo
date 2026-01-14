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

// Package files provides utilities for reading and updating version files.
package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ReadVersion reads the version from a VERSION file.
func ReadVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", fmt.Errorf("VERSION file is empty")
	}

	return version, nil
}

// WriteVersion writes a version to a VERSION file.
func WriteVersion(path, version string) error {
	// Ensure version ends with newline
	content := strings.TrimSpace(version) + "\n"

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// UpdateChartYAML updates the version and appVersion in a Chart.yaml file.
func UpdateChartYAML(chartPath, version string) error {
	chartFile := filepath.Join(chartPath, "Chart.yaml")

	// Read existing file
	data, err := os.ReadFile(chartFile)
	if err != nil {
		return fmt.Errorf("reading Chart.yaml: %w", err)
	}

	// Parse YAML while preserving structure
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return fmt.Errorf("parsing Chart.yaml: %w", err)
	}

	// Update version and appVersion fields
	if err := updateYAMLField(&node, "version", version); err != nil {
		return fmt.Errorf("updating version field: %w", err)
	}

	if err := updateYAMLField(&node, "appVersion", version); err != nil {
		return fmt.Errorf("updating appVersion field: %w", err)
	}

	// Write back
	out, err := yaml.Marshal(&node)
	if err != nil {
		return fmt.Errorf("marshaling Chart.yaml: %w", err)
	}

	if err := os.WriteFile(chartFile, out, 0644); err != nil {
		return fmt.Errorf("writing Chart.yaml: %w", err)
	}

	return nil
}

// UpdateValuesYAML updates the image.tag in a values.yaml file.
func UpdateValuesYAML(chartPath, version string) error {
	valuesFile := filepath.Join(chartPath, "values.yaml")

	// Read existing file
	data, err := os.ReadFile(valuesFile)
	if err != nil {
		return fmt.Errorf("reading values.yaml: %w", err)
	}

	// Parse YAML while preserving structure
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return fmt.Errorf("parsing values.yaml: %w", err)
	}

	// Update image.tag field (with 'v' prefix for container images)
	tagValue := "v" + version
	if err := updateNestedYAMLField(&node, []string{"image", "tag"}, tagValue); err != nil {
		return fmt.Errorf("updating image.tag field: %w", err)
	}

	// Write back
	out, err := yaml.Marshal(&node)
	if err != nil {
		return fmt.Errorf("marshaling values.yaml: %w", err)
	}

	if err := os.WriteFile(valuesFile, out, 0644); err != nil {
		return fmt.Errorf("writing values.yaml: %w", err)
	}

	return nil
}

// updateYAMLField updates a top-level field in a YAML node.
func updateYAMLField(node *yaml.Node, key, value string) error {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return updateYAMLField(node.Content[0], key, value)
	}

	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node")
	}

	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content[i+1].Value = value
			return nil
		}
	}

	return fmt.Errorf("field %s not found", key)
}

// updateNestedYAMLField updates a nested field in a YAML node.
func updateNestedYAMLField(node *yaml.Node, path []string, value string) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return updateNestedYAMLField(node.Content[0], path, value)
	}

	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node")
	}

	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == path[0] {
			if len(path) == 1 {
				// Found the target field
				node.Content[i+1].Value = value
				return nil
			}
			// Recurse into nested structure
			return updateNestedYAMLField(node.Content[i+1], path[1:], value)
		}
	}

	return fmt.Errorf("field %s not found", path[0])
}
