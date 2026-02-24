//go:build windows

package platform

import (
	"os"
	"path/filepath"
)

func DataDir() string {
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		return filepath.Join(appData, "Idra")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".idra")
}

func ConfigDir() string {
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		return filepath.Join(appData, "Idra")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".idra")
}
