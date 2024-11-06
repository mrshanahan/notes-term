package paths

import (
	"os"
	"path/filepath"
)

func EnsureLocalCacheFolder() (string, error) {
	var path = filepath.Join(os.Getenv("HOME"), ".notes-term")
	if err := os.MkdirAll(path, 0770); err != nil {
		return "", err
	}
	return path, nil
}
