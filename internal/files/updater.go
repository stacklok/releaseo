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

	// Update version field using surgical replacement
	if err := UpdateYAMLFile(VersionFileConfig{File: chartFile, Path: "version"}, version); err != nil {
		return fmt.Errorf("updating version field: %w", err)
	}

	// Update appVersion field using surgical replacement
	if err := UpdateYAMLFile(VersionFileConfig{File: chartFile, Path: "appVersion"}, version); err != nil {
		return fmt.Errorf("updating appVersion field: %w", err)
	}

	return nil
}

// UpdateValuesYAML updates the image.tag in a values.yaml file.
func UpdateValuesYAML(chartPath, version string) error {
	valuesFile := filepath.Join(chartPath, "values.yaml")

	// Update image.tag field with 'v' prefix for container images
	if err := UpdateYAMLFile(VersionFileConfig{File: valuesFile, Path: "image.tag", Prefix: "v"}, version); err != nil {
		return fmt.Errorf("updating image.tag field: %w", err)
	}

	return nil
}
