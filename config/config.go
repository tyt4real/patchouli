package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

const ConfigFileName = "patchy.json"

type BoardConfig struct {
	Name    string `json:"name"`
	SiteURL string `json:"site_url"`
}

type Config struct {
	Boards          []BoardConfig `json:"boards"`
	CooldownSeconds int           `json:"cooldown_seconds"`
}

func LoadConfig() (*Config, error) {
	configPath := ConfigFileName

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config JSON: %w", err)
	}

	return &cfg, nil
}
