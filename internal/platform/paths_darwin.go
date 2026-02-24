//go:build darwin

package platform

import (
	"os"
	"path/filepath"
)

func DataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".idra")
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".idra")
}
