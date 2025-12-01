package main

import (
	"path/filepath"
)

// normalizeDirectoryPath converts directory paths to use forward slashes
// for consistency with how paths are stored in the database.
// This matches the normalization in internal/interceptor/capture.go
func normalizeDirectoryPath(dir string) string {
	return filepath.ToSlash(filepath.Clean(dir))
}
