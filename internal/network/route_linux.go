//go:build linux

package network

import (
	"bytes"
	"context"
	"fmt"
	"log"
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

	primaryRoutes, duplicateRoutes := dedupeDefaultRoutes(routes)
	for _, route := range duplicateRoutes {
		if err := deleteDefaultRoute(ctx, route); err != nil {
			return fmt.Errorf("cleanup duplicate default route for %s: %w", route.Dev, err)
		}
	}

	_, plan, err := metricPlan(primaryRoutes, iface)
	if err != nil {
		return err
	}

	sort.Slice(primaryRoutes, func(i, j int) bool {
		if primaryRoutes[i].Dev == iface {
			return true
		}
		if primaryRoutes[j].Dev == iface {
			return false
		}
		return primaryRoutes[i].Metric < primaryRoutes[j].Metric
	})

	for _, route := range primaryRoutes {
		if err := replaceDefaultRouteMetric(ctx, route, plan[route.Dev]); err != nil {
			return err
		}
	}
	return nil
}

func (RouteSwitcher) VerifyDefaultInterface(ctx context.Context, iface string) error {
	routes, err := queryDefaultRoutes(ctx)
	if err != nil {
		return err
	}

	active, err := selectActiveDefaultRoute(routes)
	if err != nil {
		return err
	}
	if active.Dev != iface {
		return fmt.Errorf("expected active default route on %s, got %s; routes=%s", iface, active.Dev, formatDefaultRoutes(routes))
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
	if route.Metric == metric {
		return nil
	}

	// After duplicate routes are cleaned up, replacing the remaining route in
	// place is safer than add+del: it keeps all original attributes while only
	// changing metric, and avoids deleting the freshly updated route by mistake.
	args := append([]string{"route", "replace", "default"}, route.attributesWithMetric(metric)...)
	log.Printf("update default route dev=%s old_metric=%d new_metric=%d cmd=%q", route.Dev, route.Metric, metric, "ip "+strings.Join(args, " "))
	if err := runIPRouteCommand(ctx, args); err != nil {
		return fmt.Errorf("replace default route for %s: %w", route.Dev, err)
	}
	return nil
}

func deleteDefaultRoute(ctx context.Context, route defaultRoute) error {
	args := append([]string{"route", "del", "default"}, route.attributesWithMetric(route.Metric)...)
	log.Printf("cleanup duplicate default route dev=%s metric=%d cmd=%q", route.Dev, route.Metric, "ip "+strings.Join(args, " "))
	if err := runIPRouteCommand(ctx, args); err != nil {
		return fmt.Errorf("delete default route %s metric %d: %w", route.Dev, route.Metric, err)
	}
	return nil
}

func runIPRouteCommand(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "ip", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ip %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
