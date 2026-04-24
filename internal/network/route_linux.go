//go:build linux

package network

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

type RouteSwitcher struct{}

func (RouteSwitcher) CurrentDefaultInterface(ctx context.Context) (string, error) {
	routes, err := queryDefaultRoutes(ctx)
	if err != nil {
		return "", err
	}

	active, err := selectActiveDefaultRoute(routes)
	if err != nil {
		return "", err
	}
	return active.Dev, nil
}

func (RouteSwitcher) SwitchDefaultInterface(ctx context.Context, iface string) error {
	routes, err := queryDefaultRoutes(ctx)
	if err != nil {
		return err
	}

	_, plan, err := metricPlan(routes, iface)
	if err != nil {
		return err
	}

	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Dev == iface {
			return true
		}
		if routes[j].Dev == iface {
			return false
		}
		return routes[i].Metric < routes[j].Metric
	})

	for _, route := range routes {
		if err := replaceDefaultRouteMetric(ctx, route, plan[route.Dev]); err != nil {
			return err
		}
	}
	return nil
}

func queryDefaultRoutes(ctx context.Context) ([]defaultRoute, error) {
	cmd := exec.CommandContext(ctx, "ip", "route", "show", "default")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("query default route: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	routes, err := parseDefaultRoutes(stdout.String())
	if err != nil {
		return nil, err
	}
	return routes, nil
}

func replaceDefaultRouteMetric(ctx context.Context, route defaultRoute, metric int) error {
	args := []string{"route", "replace", "default"}
	if route.Gateway != "" {
		args = append(args, "via", route.Gateway)
	}
	args = append(args, "dev", route.Dev, "metric", fmt.Sprintf("%d", metric))

	cmd := exec.CommandContext(ctx, "ip", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("replace default route for %s: %w: %s", route.Dev, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
