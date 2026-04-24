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
	if routes[1].Dev != "usb1" || routes[1].Gateway != "192.168.42.1" || routes[1].Metric != 200 {
		t.Fatalf("routes[1] = %#v", routes[1])
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
