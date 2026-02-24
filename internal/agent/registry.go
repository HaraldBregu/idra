package agent

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// Registry holds discovered agent manifests and a skill→agent lookup map.
type Registry struct {
	agents   []Manifest
	skillMap map[string]string // skill name → agent name
	baseDir  string           // project root for resolving relative paths
}

// NewRegistry creates a registry by scanning the given agents directory
// for subdirectories containing manifest.json files.
func NewRegistry(agentsDir string) (*Registry, error) {
	baseDir := filepath.Dir(agentsDir) // project root is parent of agents/
	r := &Registry{
		skillMap: make(map[string]string),
		baseDir:  baseDir,
	}

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("agents directory not found, no agents to load", "dir", agentsDir)
			return r, nil
		}
		return nil, fmt.Errorf("read agents dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(agentsDir, entry.Name(), "manifest.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue
		}

		m, err := LoadManifest(manifestPath)
		if err != nil {
			slog.Warn("skipping agent", "dir", entry.Name(), "error", err)
			continue
		}

		// Register skills
		for _, skill := range m.Skills {
			if existing, ok := r.skillMap[skill]; ok {
				slog.Warn("skill conflict, keeping first agent",
					"skill", skill, "kept", existing, "skipped", m.Name)
				continue
			}
			r.skillMap[skill] = m.Name
		}

		r.agents = append(r.agents, m)
		slog.Info("registered agent", "name", m.Name, "skills", m.Skills)
	}

	return r, nil
}

// Agents returns all registered manifests.
func (r *Registry) Agents() []Manifest {
	return r.agents
}

// AgentForSkill returns the agent name that handles the given skill.
func (r *Registry) AgentForSkill(skill string) (string, bool) {
	name, ok := r.skillMap[skill]
	return name, ok
}

// BaseDir returns the project root directory.
func (r *Registry) BaseDir() string {
	return r.baseDir
}
