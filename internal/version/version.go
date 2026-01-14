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

// Package version provides semantic version parsing and manipulation.
package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a semantic version.
type Version struct {
	Major int
	Minor int
	Patch int
}

// Parse parses a semantic version string.
func Parse(s string) (*Version, error) {
	// Remove 'v' prefix if present
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimSpace(s)

	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid version format: %s (expected MAJOR.MINOR.PATCH)", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	if major < 0 || minor < 0 || patch < 0 {
		return nil, fmt.Errorf("version components cannot be negative")
	}

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// String returns the version as a string.
func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Bump returns a new version with the specified component bumped.
func (v *Version) Bump(bumpType string) (*Version, error) {
	switch strings.ToLower(bumpType) {
	case "major":
		return &Version{
			Major: v.Major + 1,
			Minor: 0,
			Patch: 0,
		}, nil
	case "minor":
		return &Version{
			Major: v.Major,
			Minor: v.Minor + 1,
			Patch: 0,
		}, nil
	case "patch":
		return &Version{
			Major: v.Major,
			Minor: v.Minor,
			Patch: v.Patch + 1,
		}, nil
	default:
		return nil, fmt.Errorf("invalid bump type: %s (expected major, minor, or patch)", bumpType)
	}
}

// Compare compares two versions.
// Returns -1 if v < other, 0 if v == other, 1 if v > other.
func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}

	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}

	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	return 0
}

// IsGreater returns true if version a is greater than version b.
func IsGreater(a, b string) bool {
	va, err := Parse(a)
	if err != nil {
		return false
	}

	vb, err := Parse(b)
	if err != nil {
		return false
	}

	return va.Compare(vb) > 0
}
