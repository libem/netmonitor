package network

import "testing"

func TestParseDefaultRoutes(t *testing.T) {
	t.Parallel()

	output := `default via 192.168.0.1 dev eth0 proto dhcp metric 100
default via 192.168.42.1 dev usb1 proto dhcp metric 200
10.252.3.0/24 dev wg0 scope link`

	routes, err := parseDefaultRoutes(output)
	if err != nil {
		t.Fatalf("parseDefaultRoutes() error = %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2", len(routes))
	}
	if routes[0].Dev != "eth0" || routes[0].Gateway != "192.168.0.1" || routes[0].Metric != 100 {
		t.Fatalf("routes[0] = %#v", routes[0])
	}
	if got, want := routes[0].Attributes, []string{"via", "192.168.0.1", "dev", "eth0", "proto", "dhcp", "metric", "100"}; len(got) != len(want) {
		t.Fatalf("routes[0].Attributes = %#v, want %#v", got, want)
	}
	if routes[1].Dev != "usb1" || routes[1].Gateway != "192.168.42.1" || routes[1].Metric != 200 {
		t.Fatalf("routes[1] = %#v", routes[1])
	}
}

func TestDefaultRouteAttributesWithMetricPreservesOtherFields(t *testing.T) {
	t.Parallel()

	route := defaultRoute{
		Dev:        "usb1",
		Gateway:    "192.168.42.1",
		Metric:     200,
		Attributes: []string{"via", "192.168.42.1", "dev", "usb1", "proto", "dhcp", "metric", "200"},
	}

	got := route.attributesWithMetric(100)
	want := []string{"via", "192.168.42.1", "dev", "usb1", "proto", "dhcp", "metric", "100"}
	if len(got) != len(want) {
		t.Fatalf("attributesWithMetric() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("attributesWithMetric()[%d] = %q, want %q; full=%#v", i, got[i], want[i], got)
		}
	}
}

func TestSelectActiveDefaultRoute(t *testing.T) {
	t.Parallel()

	active, err := selectActiveDefaultRoute([]defaultRoute{{Dev: "usb1", Metric: 200}, {Dev: "eth0", Metric: 100}})
	if err != nil {
		t.Fatalf("selectActiveDefaultRoute() error = %v", err)
	}
	if active.Dev != "eth0" {
		t.Fatalf("active.Dev = %s, want eth0", active.Dev)
	}
}

func TestMetricPlanPrefersTargetInterface(t *testing.T) {
	t.Parallel()

	routes := []defaultRoute{
		{Dev: "eth0", Gateway: "192.168.0.1", Metric: 100},
		{Dev: "usb1", Gateway: "192.168.42.1", Metric: 200},
	}

	preferredMetric, plan, err := metricPlan(routes, "usb1")
	if err != nil {
		t.Fatalf("metricPlan() error = %v", err)
	}
	if preferredMetric != 100 {
		t.Fatalf("preferredMetric = %d, want 100", preferredMetric)
	}
	if got := plan["usb1"]; got != 100 {
		t.Fatalf("plan[usb1] = %d, want 100", got)
	}
	if got := plan["eth0"]; got != 300 {
		t.Fatalf("plan[eth0] = %d, want 300", got)
	}
}

func TestMetricPlanRejectsUnknownInterface(t *testing.T) {
	t.Parallel()

	_, _, err := metricPlan([]defaultRoute{{Dev: "eth0", Metric: 100}}, "usb1")
	if err == nil {
		t.Fatal("metricPlan() error = nil, want error")
	}
}

func TestDedupeDefaultRoutes(t *testing.T) {
	t.Parallel()

	routes := []defaultRoute{
		{Dev: "eth0", Gateway: "192.168.0.1", Metric: 300},
		{Dev: "usb1", Gateway: "192.168.42.1", Metric: 100},
		{Dev: "eth0", Gateway: "192.168.0.1", Metric: 100},
		{Dev: "usb1", Gateway: "192.168.42.1", Metric: 200},
	}

	primary, duplicates := dedupeDefaultRoutes(routes)
	if len(primary) != 2 {
		t.Fatalf("len(primary) = %d, want 2", len(primary))
	}
	if primary[0].Dev != "eth0" || primary[0].Metric != 100 {
		t.Fatalf("primary[0] = %#v, want eth0 metric 100", primary[0])
	}
	if primary[1].Dev != "usb1" || primary[1].Metric != 100 {
		t.Fatalf("primary[1] = %#v, want usb1 metric 100", primary[1])
	}
	if len(duplicates) != 2 {
		t.Fatalf("len(duplicates) = %d, want 2", len(duplicates))
	}
}

func TestFormatDefaultRoutes(t *testing.T) {
	t.Parallel()

	got := formatDefaultRoutes([]defaultRoute{
		{Dev: "eth0", Gateway: "192.168.0.1", Metric: 200},
		{Dev: "usb1", Gateway: "192.168.42.1", Metric: 600},
	})
	want := "eth0(via=192.168.0.1,metric=200), usb1(via=192.168.42.1,metric=600)"
	if got != want {
		t.Fatalf("formatDefaultRoutes() = %q, want %q", got, want)
	}
}
