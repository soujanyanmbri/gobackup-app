package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if strings.Contains(absPath, "..") {
		return fmt.Errorf("directory traversal not allowed")
	}

	return nil
}

func EnsureDirectoryExists(dirPath string) error {
	if err := ValidatePath(dirPath); err != nil {
		return err
	}

	return os.MkdirAll(dirPath, 0755)
}

func GetFileInfo(filePath string) (os.FileInfo, error) {
	return os.Stat(filePath)
}

func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
