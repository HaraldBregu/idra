package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"idra/internal/platform"
)

// AgentConfig overrides for an individual agent (optional, for future use).
type AgentConfig struct {
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled,omitempty"` // nil = true (default enabled)
}

type Config struct {
	Port        int           `json:"port"`
	BearerToken string        `json:"bearer_token"`
	AutoOpen    bool          `json:"auto_open_browser"`
	Agents      []AgentConfig `json:"agents,omitempty"`
}

func Default() Config {
	return Config{
		Port:       8080,
		BearerToken: "",
		AutoOpen:   true,
	}
}

var (
	mu       sync.RWMutex
	current  Config
	filePath string
)

func init() {
	filePath = filepath.Join(platform.ConfigDir(), "config.json")
}

func FilePath() string { return filePath }

func Load() (Config, error) {
	mu.Lock()
	defer mu.Unlock()

	current = Default()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			current.BearerToken = generateToken()
			return current, save()
		}
		return current, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, &current); err != nil {
		return current, fmt.Errorf("parse config: %w", err)
	}

	if current.BearerToken == "" {
		current.BearerToken = generateToken()
		if err := save(); err != nil {
			return current, err
		}
	}

	return current, nil
}

func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

func Update(fn func(*Config)) (Config, error) {
	mu.Lock()
	defer mu.Unlock()

	fn(&current)
	if err := validate(current); err != nil {
		return current, err
	}
	return current, save()
}

func Replace(c Config) (Config, error) {
	mu.Lock()
	defer mu.Unlock()

	// Preserve bearer token — it cannot be changed via API
	c.BearerToken = current.BearerToken
	if err := validate(c); err != nil {
		return current, err
	}
	current = c
	return current, save()
}

// save writes current config to disk. Caller must hold mu.
func save() error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmp := filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp, filePath); err != nil {
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

func validate(c Config) error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", c.Port)
	}
	return nil
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
