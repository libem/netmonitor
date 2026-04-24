package app

import (
	"testing"

	"netmonitor/internal/evaluator"
	"netmonitor/internal/monitor"
)

func TestFindScore(t *testing.T) {
	t.Parallel()

	scores := []evaluator.InterfaceScore{{Name: "lan0", Score: 10}, {Name: "wwan0", Score: 20}}

	score, ok := findScore(scores, "wwan0")
	if !ok {
		t.Fatal("findScore() ok = false, want true")
	}
	if score.Name != "wwan0" {
		t.Fatalf("findScore() name = %s, want wwan0", score.Name)
	}
}

func TestFormatPingResult(t *testing.T) {
	t.Parallel()

	formatted := formatPingResult(monitor.PingResult{Success: true, PacketLoss: 0, AverageRTTMS: 18.5})
	want := "success loss=0.00% avg_rtt=18.50ms"
	if formatted != want {
		t.Fatalf("formatPingResult() = %q, want %q", formatted, want)
	}
}

func TestFormatScoreRanking(t *testing.T) {
	t.Parallel()

	ranking := formatScoreRanking([]evaluator.InterfaceScore{{Name: "lan0", Score: 90.1}, {Name: "wwan0", Score: 77.3}})
	want := "lan0(90.10) > wwan0(77.30)"
	if ranking != want {
		t.Fatalf("formatScoreRanking() = %q, want %q", ranking, want)
	}
}

func TestDetectInterfaceChanges(t *testing.T) {
	t.Parallel()

	added, removed := detectInterfaceChanges([]string{"eth0"}, []string{"eth0", "usb1"})
	if len(added) != 1 || added[0] != "usb1" {
		t.Fatalf("added = %#v, want [usb1]", added)
	}
	if len(removed) != 0 {
		t.Fatalf("removed = %#v, want []", removed)
	}

	added, removed = detectInterfaceChanges([]string{"eth0", "usb1"}, []string{"eth0"})
	if len(added) != 0 {
		t.Fatalf("added = %#v, want []", added)
	}
	if len(removed) != 1 || removed[0] != "usb1" {
		t.Fatalf("removed = %#v, want [usb1]", removed)
	}
}
