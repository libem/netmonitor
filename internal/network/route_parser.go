package network

import (
	"fmt"
	"strconv"
	"strings"
)

type defaultRoute struct {
	Gateway string
	Dev     string
	Metric  int
	Raw     string
}

func parseDefaultRoutes(output string) ([]defaultRoute, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	routes := make([]defaultRoute, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "default" {
			continue
		}

		route := defaultRoute{Metric: 0, Raw: line}
		for i := 1; i < len(fields); i++ {
			switch fields[i] {
			case "via":
				if i+1 >= len(fields) {
					return nil, fmt.Errorf("default route missing gateway: %s", line)
				}
				route.Gateway = fields[i+1]
				i++
			case "dev":
				if i+1 >= len(fields) {
					return nil, fmt.Errorf("default route missing device: %s", line)
				}
				route.Dev = fields[i+1]
				i++
			case "metric":
				if i+1 >= len(fields) {
					return nil, fmt.Errorf("default route missing metric: %s", line)
				}
				metric, err := strconv.Atoi(fields[i+1])
				if err != nil {
					return nil, fmt.Errorf("parse route metric from %q: %w", line, err)
				}
				route.Metric = metric
				i++
			}
		}
		if route.Dev == "" {
			return nil, fmt.Errorf("default route missing device: %s", line)
		}
		routes = append(routes, route)
	}

	if len(routes) == 0 {
		return nil, fmt.Errorf("no default routes found")
	}
	return routes, nil
}

func selectActiveDefaultRoute(routes []defaultRoute) (defaultRoute, error) {
	if len(routes) == 0 {
		return defaultRoute{}, fmt.Errorf("no default routes available")
	}
	active := routes[0]
	for _, route := range routes[1:] {
		if route.Metric < active.Metric {
			active = route
		}
	}
	return active, nil
}

func metricPlan(routes []defaultRoute, preferredDev string) (int, map[string]int, error) {
	if len(routes) == 0 {
		return 0, nil, fmt.Errorf("no default routes available")
	}

	preferredMetric := routes[0].Metric
	maxMetric := routes[0].Metric
	found := false
	for _, route := range routes {
		if route.Metric < preferredMetric {
			preferredMetric = route.Metric
		}
		if route.Metric > maxMetric {
			maxMetric = route.Metric
		}
		if route.Dev == preferredDev {
			found = true
		}
	}
	if !found {
		return 0, nil, fmt.Errorf("target interface %s has no default route", preferredDev)
	}

	fallbackMetric := maxMetric + 100
	if fallbackMetric <= preferredMetric {
		fallbackMetric = preferredMetric + 100
	}

	plan := make(map[string]int, len(routes))
	shift := 0
	for _, route := range routes {
		if route.Dev == preferredDev {
			plan[route.Dev] = preferredMetric
			continue
		}
		plan[route.Dev] = fallbackMetric + shift
		shift += 100
	}

	return preferredMetric, plan, nil
}
