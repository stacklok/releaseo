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
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// VersionFileConfig defines a YAML file and the path to update with the new version.
type VersionFileConfig struct {
	File   string `json:"file"`
	Path   string `json:"path"`
	Prefix string `json:"prefix,omitempty"`
}

// UpdateYAMLFile updates a specific path in a YAML file with a new version.
func UpdateYAMLFile(cfg VersionFileConfig, version string) error {
	data, err := os.ReadFile(cfg.File)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", cfg.File, err)
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return fmt.Errorf("parsing YAML %s: %w", cfg.File, err)
	}

	// Apply prefix if specified
	value := version
	if cfg.Prefix != "" {
		value = cfg.Prefix + version
	}

	// Parse the path and update the node
	pathParts, err := parsePath(cfg.Path)
	if err != nil {
		return fmt.Errorf("parsing path %s: %w", cfg.Path, err)
	}

	if err := updateNodeAtPath(&node, pathParts, value); err != nil {
		return fmt.Errorf("updating path %s in %s: %w", cfg.Path, cfg.File, err)
	}

	// Write back
	out, err := yaml.Marshal(&node)
	if err != nil {
		return fmt.Errorf("marshaling YAML %s: %w", cfg.File, err)
	}

	if err := os.WriteFile(cfg.File, out, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", cfg.File, err)
	}

	return nil
}

// pathPart represents a single segment of a YAML path.
type pathPart struct {
	Key   string
	Index int // -1 if not an array index
}

// parsePath parses a dot-notation path with optional array indices.
// Examples: "image.tag", "spec.containers[0].image", "metadata.labels[\"app.kubernetes.io/version\"]"
func parsePath(path string) ([]pathPart, error) {
	var parts []pathPart

	// Regex to match: key, key[0], key["string"], key['string']
	segmentRegex := regexp.MustCompile(`([^.\[\]]+)(?:\[(\d+|"[^"]*"|'[^']*')\])?`)

	// Split by dots, but handle the segments properly
	remaining := path
	for remaining != "" {
		// Find the next segment
		match := segmentRegex.FindStringSubmatchIndex(remaining)
		if match == nil {
			return nil, fmt.Errorf("invalid path segment in: %s", remaining)
		}

		key := remaining[match[2]:match[3]]
		part := pathPart{Key: key, Index: -1}

		// Check for array index
		if match[4] != -1 && match[5] != -1 {
			indexStr := remaining[match[4]:match[5]]
			// Check if it's a numeric index
			if idx, err := strconv.Atoi(indexStr); err == nil {
				part.Index = idx
			} else {
				// It's a quoted string key - strip quotes and treat as a key
				indexStr = strings.Trim(indexStr, `"'`)
				// Add the array access as a separate part
				parts = append(parts, part)
				part = pathPart{Key: indexStr, Index: -1}
			}
		}

		parts = append(parts, part)

		// Move past this segment
		remaining = remaining[match[1]:]
		remaining = strings.TrimPrefix(remaining, ".")
	}

	return parts, nil
}

// updateNodeAtPath updates a value at the specified path in a YAML node.
func updateNodeAtPath(node *yaml.Node, path []pathPart, value string) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	// Handle document node
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return fmt.Errorf("empty document")
		}
		return updateNodeAtPath(node.Content[0], path, value)
	}

	current := path[0]
	remaining := path[1:]

	//nolint:exhaustive // We only handle the node types we expect
	switch node.Kind {
	case yaml.MappingNode:
		// Find the key in the mapping
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == current.Key {
				valueNode := node.Content[i+1]

				// If there's an array index, handle it
				if current.Index >= 0 {
					if valueNode.Kind != yaml.SequenceNode {
						return fmt.Errorf("expected sequence at %s, got %v", current.Key, valueNode.Kind)
					}
					if current.Index >= len(valueNode.Content) {
						return fmt.Errorf("index %d out of bounds for %s (len=%d)", current.Index, current.Key, len(valueNode.Content))
					}
					valueNode = valueNode.Content[current.Index]
				}

				// If this is the last part, update the value
				if len(remaining) == 0 {
					valueNode.Value = value
					return nil
				}

				// Otherwise, recurse
				return updateNodeAtPath(valueNode, remaining, value)
			}
		}
		return fmt.Errorf("key %s not found", current.Key)

	case yaml.SequenceNode:
		if current.Index < 0 {
			return fmt.Errorf("expected array index for sequence node")
		}
		if current.Index >= len(node.Content) {
			return fmt.Errorf("index %d out of bounds (len=%d)", current.Index, len(node.Content))
		}

		if len(remaining) == 0 {
			node.Content[current.Index].Value = value
			return nil
		}

		return updateNodeAtPath(node.Content[current.Index], remaining, value)

	default:
		return fmt.Errorf("unexpected node kind: %v", node.Kind)
	}
}
