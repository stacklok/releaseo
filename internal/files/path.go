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
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath ensures a file path is safe and within the allowed base directory.
// It prevents path traversal attacks by checking that the resolved path stays within bounds.
// If basePath is empty, the current working directory is used.
func ValidatePath(basePath, userPath string) (string, error) {
	if userPath == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Get absolute base path
	if basePath == "" {
		var err error
		basePath, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting working directory: %w", err)
		}
	}

	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", fmt.Errorf("resolving base path: %w", err)
	}

	// Clean and resolve the user path
	cleanPath := filepath.Clean(userPath)

	// Check for obvious path traversal attempts
	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "/../") {
		return "", fmt.Errorf("path traversal detected in %q", userPath)
	}

	// Resolve the full path
	var fullPath string
	if filepath.IsAbs(cleanPath) {
		fullPath = cleanPath
	} else {
		fullPath = filepath.Join(absBase, cleanPath)
	}

	// Get absolute path to handle any remaining relative components
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	// Ensure the resolved path is within the base directory
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return "", fmt.Errorf("path %q resolves outside allowed directory", userPath)
	}

	return absPath, nil
}

// ValidatePathRelative validates a path and returns it relative to the base directory.
// This is useful when the relative path is needed for display or storage.
func ValidatePathRelative(basePath, userPath string) (string, error) {
	absPath, err := ValidatePath(basePath, userPath)
	if err != nil {
		return "", err
	}

	// Get absolute base path for relative calculation
	if basePath == "" {
		basePath, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting working directory: %w", err)
		}
	}

	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", fmt.Errorf("resolving base path: %w", err)
	}

	relPath, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return "", fmt.Errorf("calculating relative path: %w", err)
	}

	return relPath, nil
}
