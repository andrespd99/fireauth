package config

import (
	"os"
	"path/filepath"

	"github.com/cashea-bnpl/auth-devtools/internal/logger"
)

const dirName = ".cashea-auth"

// Dir returns the absolute path to the configuration directory (~/.cashea-auth).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, dirName), nil
}

// EnsureDir creates the configuration directory with 0700 permissions if it
// does not already exist.
func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	logger.Debug("ensuring config directory exists", "path", dir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// FilePath returns the full path to a file inside the config directory.
func FilePath(name string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}
