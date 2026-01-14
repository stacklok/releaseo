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
	"os"
	"testing"
)

// createTempFile creates a temporary file with the given content and returns its path.
// The file is automatically cleaned up when the test completes.
func createTempFile(t *testing.T, content string, pattern string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	if content != "" {
		if err := os.WriteFile(tmpFile.Name(), []byte(content), 0600); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
	}
	tmpFile.Close()

	return tmpFile.Name()
}

// readTempFile reads and returns the content of a file.
func readTempFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	return string(content)
}
