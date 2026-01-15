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

// VersionReader reads version information from files.
type VersionReader interface {
	// ReadVersion reads the version from the specified path.
	ReadVersion(path string) (string, error)
}

// VersionWriter writes version information to files.
type VersionWriter interface {
	// WriteVersion writes the version to the specified path.
	WriteVersion(path, version string) error
}

// YAMLUpdater updates version information in YAML files.
type YAMLUpdater interface {
	// UpdateYAMLFile updates a specific path in a YAML file with a new version.
	UpdateYAMLFile(cfg VersionFileConfig, currentVersion, newVersion string) error
}

// DefaultVersionReader is the default implementation of VersionReader.
type DefaultVersionReader struct{}

// ReadVersion reads the version from the specified path.
func (*DefaultVersionReader) ReadVersion(path string) (string, error) {
	return ReadVersion(path)
}

// DefaultVersionWriter is the default implementation of VersionWriter.
type DefaultVersionWriter struct{}

// WriteVersion writes the version to the specified path.
func (*DefaultVersionWriter) WriteVersion(path, version string) error {
	return WriteVersion(path, version)
}

// DefaultYAMLUpdater is the default implementation of YAMLUpdater.
type DefaultYAMLUpdater struct{}

// UpdateYAMLFile updates a specific path in a YAML file with a new version.
func (*DefaultYAMLUpdater) UpdateYAMLFile(cfg VersionFileConfig, currentVersion, newVersion string) error {
	return UpdateYAMLFile(cfg, currentVersion, newVersion)
}
