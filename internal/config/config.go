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

// ProjectsDir returns the path to the projects directory (~/.cashea-auth/projects).
func ProjectsDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "projects"), nil
}

// ProjectDir returns the path to a specific project's directory.
func ProjectDir(name string) (string, error) {
	pdir, err := ProjectsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(pdir, name), nil
}

// ProjectFilePath returns the full path to a file inside a project's directory.
func ProjectFilePath(projectName, filename string) (string, error) {
	pdir, err := ProjectDir(projectName)
	if err != nil {
		return "", err
	}
	return filepath.Join(pdir, filename), nil
}
