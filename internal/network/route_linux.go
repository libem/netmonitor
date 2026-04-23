//go:build linux

package network

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type RouteSwitcher struct{}

func (RouteSwitcher) CurrentDefaultInterface(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "ip", "route", "show", "default")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("query default route: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	fields := strings.Fields(stdout.String())
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] == "dev" {
			return fields[i+1], nil
		}
	}

	return "", fmt.Errorf("default route interface not found")
}

func (RouteSwitcher) SwitchDefaultInterface(ctx context.Context, iface string) error {
	cmd := exec.CommandContext(ctx, "ip", "route", "replace", "default", "dev", iface)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("switch default route to %s: %w: %s", iface, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
