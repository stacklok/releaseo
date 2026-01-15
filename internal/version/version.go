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
	if c := cmpInt(v.Major, other.Major); c != 0 {
		return c
	}
	if c := cmpInt(v.Minor, other.Minor); c != 0 {
		return c
	}
	return cmpInt(v.Patch, other.Patch)
}

// cmpInt compares two integers and returns -1, 0, or 1.
func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// CompareVersions compares two version strings and returns their relative ordering.
// It returns -1 if a < b, 0 if a == b, and 1 if a > b.
// If either version string cannot be parsed, an error is returned with context
// indicating which version failed to parse.
func CompareVersions(a, b string) (int, error) {
	va, err := Parse(a)
	if err != nil {
		return 0, fmt.Errorf("parsing version a %q: %w", a, err)
	}

	vb, err := Parse(b)
	if err != nil {
		return 0, fmt.Errorf("parsing version b %q: %w", b, err)
	}

	return va.Compare(vb), nil
}

// IsGreaterE returns true if version a is greater than version b, along with
// any error that occurred during parsing. This is the preferred function for
// new code as it allows callers to distinguish between "version A is not greater"
// and "invalid version string".
//
// If an error is returned, the boolean result should be ignored.
func IsGreaterE(a, b string) (bool, error) {
	cmp, err := CompareVersions(a, b)
	if err != nil {
		return false, err
	}
	return cmp > 0, nil
}
