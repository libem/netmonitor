package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	PingTargets     []string
	PingIntervalSec int
	PingTimeoutSec  int
	PingCount       int
	CheckInterval   time.Duration
	Interfaces      []string
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	defer file.Close()

	cfg := Config{}
	var currentList *[]string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "- ") {
			if currentList == nil {
				return Config{}, fmt.Errorf("invalid list item without key: %s", line)
			}
			*currentList = append(*currentList, trimYAMLString(strings.TrimSpace(strings.TrimPrefix(line, "- "))))
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return Config{}, fmt.Errorf("invalid config line: %s", line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		currentList = nil

		switch key {
		case "ping_targets":
			currentList = &cfg.PingTargets
		case "interfaces":
			currentList = &cfg.Interfaces
		case "ping_interval_sec":
			cfg.PingIntervalSec, err = strconv.Atoi(value)
		case "ping_timeout_sec":
			cfg.PingTimeoutSec, err = strconv.Atoi(value)
		case "ping_count":
			cfg.PingCount, err = strconv.Atoi(value)
		case "check_interval":
			cfg.CheckInterval, err = time.ParseDuration(value)
		default:
			continue
		}

		if err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("scan config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.PingIntervalSec <= 0 {
		c.PingIntervalSec = 10
	}
}

func (c Config) Validate() error {
	if len(c.PingTargets) == 0 {
		return fmt.Errorf("config ping_targets cannot be empty")
	}
	if len(c.Interfaces) == 0 {
		return fmt.Errorf("config interfaces cannot be empty")
	}
	if c.PingTimeoutSec <= 0 {
		return fmt.Errorf("config ping_timeout_sec must be greater than 0")
	}
	if c.PingCount <= 0 {
		return fmt.Errorf("config ping_count must be greater than 0")
	}
	if c.CheckInterval <= 0 {
		return fmt.Errorf("config check_interval must be greater than 0")
	}
	return nil
}

func stripComment(line string) string {
	inQuote := false
	for i, r := range line {
		switch r {
		case '"':
			inQuote = !inQuote
		case '#':
			if !inQuote {
				return line[:i]
			}
		}
	}
	return line
}

func trimYAMLString(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "\"")
	value = strings.TrimSuffix(value, "\"")
	return value
}
