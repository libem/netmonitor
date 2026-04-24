package network

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type defaultRoute struct {
	Gateway    string
	Dev        string
	Metric     int
	Raw        string
	Attributes []string
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

		route := defaultRoute{
			Metric:     0,
			Raw:        line,
			Attributes: append([]string(nil), fields[1:]...),
		}
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

func (r defaultRoute) attributesWithMetric(metric int) []string {
	attrs := make([]string, 0, len(r.Attributes)+2)
	inserted := false
	for i := 0; i < len(r.Attributes); i++ {
		if r.Attributes[i] == "metric" {
			attrs = append(attrs, "metric", strconv.Itoa(metric))
			inserted = true
			i++
			continue
		}
		attrs = append(attrs, r.Attributes[i])
	}
	if !inserted {
		attrs = append(attrs, "metric", strconv.Itoa(metric))
	}
	return attrs
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

func dedupeDefaultRoutes(routes []defaultRoute) (primary []defaultRoute, duplicates []defaultRoute) {
	if len(routes) == 0 {
		return nil, nil
	}

	byDev := make(map[string][]defaultRoute)
	order := make([]string, 0, len(routes))
	for _, route := range routes {
		if _, ok := byDev[route.Dev]; !ok {
			order = append(order, route.Dev)
		}
		byDev[route.Dev] = append(byDev[route.Dev], route)
	}

	for _, dev := range order {
		group := byDev[dev]
		sort.SliceStable(group, func(i, j int) bool {
			return group[i].Metric < group[j].Metric
		})
		primary = append(primary, group[0])
		if len(group) > 1 {
			duplicates = append(duplicates, group[1:]...)
		}
	}

	return primary, duplicates
}

func formatDefaultRoutes(routes []defaultRoute) string {
	if len(routes) == 0 {
		return "[]"
	}

	parts := make([]string, 0, len(routes))
	for _, route := range routes {
		gateway := route.Gateway
		if gateway == "" {
			gateway = "direct"
		}
		parts = append(parts, fmt.Sprintf("%s(via=%s,metric=%d)", route.Dev, gateway, route.Metric))
	}
	return strings.Join(parts, ", ")
}
