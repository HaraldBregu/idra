//go:build linux

package platform

import (
	"os"
	"path/filepath"
)

func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "idra")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".idra")
}

func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "idra")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".idra")
}
