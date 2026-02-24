package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Manifest describes an agent loaded from its manifest.json.
type Manifest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Skills      []string `json:"skills"`
	Command     string   `json:"command"`
	Args        []string `json:"args,omitempty"`
	Dir         string   `json:"dir"` // working directory relative to project root
}

// LoadManifest reads and validates a manifest.json file.
func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest %s: %w", path, err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest %s: %w", path, err)
	}

	if err := m.Validate(); err != nil {
		return Manifest{}, fmt.Errorf("invalid manifest %s: %w", path, err)
	}

	return m, nil
}

// Validate checks required fields.
func (m Manifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(m.Skills) == 0 {
		return fmt.Errorf("at least one skill is required")
	}
	if m.Command == "" {
		return fmt.Errorf("command is required")
	}
	return nil
}

// AbsDir resolves the working directory relative to a base path.
func (m Manifest) AbsDir(base string) string {
	if filepath.IsAbs(m.Dir) {
		return m.Dir
	}
	return filepath.Join(base, m.Dir)
}
