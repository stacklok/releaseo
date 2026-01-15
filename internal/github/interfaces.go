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

package github

// FileReader defines the interface for reading file contents.
// This abstraction allows for dependency injection and makes the client
// testable by enabling mock file systems.
type FileReader interface {
	// ReadFile reads the contents of a file at the given path.
	// It returns the file contents as a byte slice, or an error if the read fails.
	ReadFile(path string) ([]byte, error)
}
