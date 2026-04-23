package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadParsesConfigWithComments(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "monitor-net.yaml")
	content := `ping_targets:
  - "8.8.8.8"
  - "www.baidu.com" # public dns
ping_interval_sec: 0
ping_timeout_sec: 10
ping_count: 5
check_interval: 15s # scheduler interval
interfaces:
  - "wwan0"
  - "lan0"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.PingIntervalSec, 10; got != want {
		t.Fatalf("PingIntervalSec = %d, want %d", got, want)
	}
	if got, want := cfg.PingTimeoutSec, 10; got != want {
		t.Fatalf("PingTimeoutSec = %d, want %d", got, want)
	}
	if got, want := cfg.PingCount, 5; got != want {
		t.Fatalf("PingCount = %d, want %d", got, want)
	}
	if got, want := cfg.CheckInterval, 15*time.Second; got != want {
		t.Fatalf("CheckInterval = %v, want %v", got, want)
	}
	if len(cfg.PingTargets) != 2 || cfg.PingTargets[1] != "www.baidu.com" {
		t.Fatalf("PingTargets = %#v, want parsed targets", cfg.PingTargets)
	}
	if len(cfg.Interfaces) != 2 || cfg.Interfaces[0] != "wwan0" || cfg.Interfaces[1] != "lan0" {
		t.Fatalf("Interfaces = %#v, want parsed interfaces", cfg.Interfaces)
	}
}

func TestLoadRejectsListItemWithoutParentKey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	content := `- "8.8.8.8"`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
