package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
// The name is cleaned with filepath.Clean to neutralize path traversal.
func ProjectDir(name string) (string, error) {
	cleaned := filepath.Clean(name)
	if err := validateProjectName(cleaned); err != nil {
		return "", err
	}
	pdir, err := ProjectsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(pdir, cleaned), nil
}

// ProjectFilePath returns the full path to a file inside a project's directory.
func ProjectFilePath(projectName, filename string) (string, error) {
	cleaned := filepath.Clean(projectName)
	if err := validateProjectName(cleaned); err != nil {
		return "", err
	}
	pdir, err := ProjectDir(cleaned)
	if err != nil {
		return "", err
	}
	return filepath.Join(pdir, filename), nil
}

// validateProjectName rejects names that could traverse outside the projects
// directory (empty, path separators, "." or ".." segments).
func validateProjectName(name string) error {
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("invalid project name %q", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid project name %q", name)
	}
	return nil
}
