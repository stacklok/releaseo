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

package version

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    *Version
		wantErr bool
	}{
		{
			name:  "basic version",
			input: "1.2.3",
			want:  &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:  "with v prefix",
			input: "v1.2.3",
			want:  &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:  "with whitespace",
			input: "  1.2.3  ",
			want:  &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:  "zero version",
			input: "0.0.0",
			want:  &Version{Major: 0, Minor: 0, Patch: 0},
		},
		{
			name:  "large numbers",
			input: "100.200.300",
			want:  &Version{Major: 100, Minor: 200, Patch: 300},
		},
		{
			name:    "invalid - too few parts",
			input:   "1.2",
			wantErr: true,
		},
		{
			name:    "invalid - too many parts",
			input:   "1.2.3.4",
			wantErr: true,
		},
		{
			name:    "invalid - non-numeric major",
			input:   "a.2.3",
			wantErr: true,
		},
		{
			name:    "invalid - non-numeric minor",
			input:   "1.b.3",
			wantErr: true,
		},
		{
			name:    "invalid - non-numeric patch",
			input:   "1.2.c",
			wantErr: true,
		},
		{
			name:    "invalid - negative major",
			input:   "-1.2.3",
			wantErr: true,
		},
		{
			name:    "invalid - empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	t.Parallel()
	v := &Version{Major: 1, Minor: 2, Patch: 3}
	if got := v.String(); got != "1.2.3" {
		t.Errorf("String() = %v, want %v", got, "1.2.3")
	}
}

func TestVersion_Bump(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  *Version
		bumpType string
		want     *Version
		wantErr  bool
	}{
		{
			name:     "bump major",
			version:  &Version{Major: 1, Minor: 2, Patch: 3},
			bumpType: "major",
			want:     &Version{Major: 2, Minor: 0, Patch: 0},
		},
		{
			name:     "bump minor",
			version:  &Version{Major: 1, Minor: 2, Patch: 3},
			bumpType: "minor",
			want:     &Version{Major: 1, Minor: 3, Patch: 0},
		},
		{
			name:     "bump patch",
			version:  &Version{Major: 1, Minor: 2, Patch: 3},
			bumpType: "patch",
			want:     &Version{Major: 1, Minor: 2, Patch: 4},
		},
		{
			name:     "bump major - uppercase",
			version:  &Version{Major: 1, Minor: 2, Patch: 3},
			bumpType: "MAJOR",
			want:     &Version{Major: 2, Minor: 0, Patch: 0},
		},
		{
			name:     "bump minor - mixed case",
			version:  &Version{Major: 1, Minor: 2, Patch: 3},
			bumpType: "Minor",
			want:     &Version{Major: 1, Minor: 3, Patch: 0},
		},
		{
			name:     "invalid bump type",
			version:  &Version{Major: 1, Minor: 2, Patch: 3},
			bumpType: "invalid",
			wantErr:  true,
		},
		{
			name:     "empty bump type",
			version:  &Version{Major: 1, Minor: 2, Patch: 3},
			bumpType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.version.Bump(tt.bumpType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bump() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
				t.Errorf("Bump() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		v     *Version
		other *Version
		want  int
	}{
		{
			name:  "equal",
			v:     &Version{Major: 1, Minor: 2, Patch: 3},
			other: &Version{Major: 1, Minor: 2, Patch: 3},
			want:  0,
		},
		{
			name:  "greater major",
			v:     &Version{Major: 2, Minor: 0, Patch: 0},
			other: &Version{Major: 1, Minor: 9, Patch: 9},
			want:  1,
		},
		{
			name:  "lesser major",
			v:     &Version{Major: 1, Minor: 9, Patch: 9},
			other: &Version{Major: 2, Minor: 0, Patch: 0},
			want:  -1,
		},
		{
			name:  "greater minor",
			v:     &Version{Major: 1, Minor: 3, Patch: 0},
			other: &Version{Major: 1, Minor: 2, Patch: 9},
			want:  1,
		},
		{
			name:  "lesser minor",
			v:     &Version{Major: 1, Minor: 2, Patch: 9},
			other: &Version{Major: 1, Minor: 3, Patch: 0},
			want:  -1,
		},
		{
			name:  "greater patch",
			v:     &Version{Major: 1, Minor: 2, Patch: 4},
			other: &Version{Major: 1, Minor: 2, Patch: 3},
			want:  1,
		},
		{
			name:  "lesser patch",
			v:     &Version{Major: 1, Minor: 2, Patch: 3},
			other: &Version{Major: 1, Minor: 2, Patch: 4},
			want:  -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.v.Compare(tt.other); got != tt.want {
				t.Errorf("Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		a       string
		b       string
		want    int
		wantErr bool
		errMsg  string
	}{
		{
			name: "a greater than b - major",
			a:    "2.0.0",
			b:    "1.0.0",
			want: 1,
		},
		{
			name: "a less than b - major",
			a:    "1.0.0",
			b:    "2.0.0",
			want: -1,
		},
		{
			name: "equal versions",
			a:    "1.0.0",
			b:    "1.0.0",
			want: 0,
		},
		{
			name: "a greater than b - minor",
			a:    "1.2.0",
			b:    "1.1.0",
			want: 1,
		},
		{
			name: "a greater than b - patch",
			a:    "1.0.2",
			b:    "1.0.1",
			want: 1,
		},
		{
			name: "with v prefix",
			a:    "v2.0.0",
			b:    "v1.0.0",
			want: 1,
		},
		{
			name:    "invalid version a",
			a:       "invalid",
			b:       "1.0.0",
			wantErr: true,
			errMsg:  "parsing version a",
		},
		{
			name:    "invalid version b",
			a:       "1.0.0",
			b:       "invalid",
			wantErr: true,
			errMsg:  "parsing version b",
		},
		{
			name:    "both versions invalid",
			a:       "bad",
			b:       "invalid",
			wantErr: true,
			errMsg:  "parsing version a",
		},
		{
			name:    "empty version a",
			a:       "",
			b:       "1.0.0",
			wantErr: true,
			errMsg:  "parsing version a",
		},
		{
			name:    "empty version b",
			a:       "1.0.0",
			b:       "",
			wantErr: true,
			errMsg:  "parsing version b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := CompareVersions(tt.a, tt.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareVersions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errMsg != "" && err != nil {
					if !strings.Contains(err.Error(), tt.errMsg) {
						t.Errorf("CompareVersions() error = %v, should contain %q", err, tt.errMsg)
					}
				}
				return
			}
			if got != tt.want {
				t.Errorf("CompareVersions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsGreaterE(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		a       string
		b       string
		want    bool
		wantErr bool
		errMsg  string
	}{
		{
			name: "greater",
			a:    "2.0.0",
			b:    "1.0.0",
			want: true,
		},
		{
			name: "lesser",
			a:    "1.0.0",
			b:    "2.0.0",
			want: false,
		},
		{
			name: "equal",
			a:    "1.0.0",
			b:    "1.0.0",
			want: false,
		},
		{
			name: "with v prefix",
			a:    "v2.0.0",
			b:    "v1.0.0",
			want: true,
		},
		{
			name:    "invalid a returns error",
			a:       "invalid",
			b:       "1.0.0",
			wantErr: true,
			errMsg:  "parsing version a",
		},
		{
			name:    "invalid b returns error",
			a:       "1.0.0",
			b:       "invalid",
			wantErr: true,
			errMsg:  "parsing version b",
		},
		{
			name:    "empty a returns error",
			a:       "",
			b:       "1.0.0",
			wantErr: true,
			errMsg:  "parsing version a",
		},
		{
			name:    "malformed version returns error",
			a:       "1.2",
			b:       "1.0.0",
			wantErr: true,
			errMsg:  "parsing version a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := IsGreaterE(tt.a, tt.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsGreaterE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errMsg != "" && err != nil {
					if !strings.Contains(err.Error(), tt.errMsg) {
						t.Errorf("IsGreaterE() error = %v, should contain %q", err, tt.errMsg)
					}
				}
				return
			}
			if got != tt.want {
				t.Errorf("IsGreaterE() = %v, want %v", got, tt.want)
			}
		})
	}
}
