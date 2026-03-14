//go:build functional

package functests

import (
	"os"
	"path/filepath"
)

func getProjectRoot() string {
	dir, _ := os.Getwd()
	return filepath.Clean(filepath.Join(dir, ".."))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(getProjectRoot(), path))
}
