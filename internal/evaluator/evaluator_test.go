package evaluator

import (
	"math"
	"testing"

	"netmonitor/internal/monitor"
)

func TestEvaluateAggregatesSamples(t *testing.T) {
	t.Parallel()

	samples := []monitor.PingResult{
		{Success: true, PacketLoss: 0, AverageRTTMS: 20},
		{Success: true, PacketLoss: 10, AverageRTTMS: 30},
		{Success: false, PacketLoss: 100, AverageRTTMS: 0},
	}

	score := (Evaluator{}).Evaluate("lan0", samples)

	if got, want := score.Reachability, 2.0/3.0; got != want {
		t.Fatalf("Reachability = %v, want %v", got, want)
	}
	if got, want := score.PacketLoss, 110.0/3.0; got != want {
		t.Fatalf("PacketLoss = %v, want %v", got, want)
	}
	if got, want := score.AverageRTTMS, 25.0; got != want {
		t.Fatalf("AverageRTTMS = %v, want %v", got, want)
	}
	if got, want := score.Score, (2.0/3.0)*100-25.0/10-(110.0/3.0)*0.7; math.Abs(got-want) > 1e-9 {
		t.Fatalf("Score = %v, want %v", got, want)
	}
}

func TestDecideSwitch(t *testing.T) {
	t.Parallel()

	best := InterfaceScore{Name: "lan0", Score: 82, Reachability: 1}
	current := InterfaceScore{Name: "wwan0", Score: 60, Reachability: 1}

	decision := DecideSwitch(best, current, true)
	if !decision.ShouldSwitch {
		t.Fatalf("ShouldSwitch = false, want true; reason=%s", decision.Reason)
	}
	if decision.ScoreDelta != 22 {
		t.Fatalf("ScoreDelta = %v, want 22", decision.ScoreDelta)
	}
}

func TestDecideSwitchKeepsCurrentWhenDeltaTooSmall(t *testing.T) {
	t.Parallel()

	best := InterfaceScore{Name: "lan0", Score: 82, Reachability: 1}
	current := InterfaceScore{Name: "wwan0", Score: 78.5, Reachability: 1}

	decision := DecideSwitch(best, current, true)
	if decision.ShouldSwitch {
		t.Fatalf("ShouldSwitch = true, want false")
	}
}

func TestDecideSwitchRejectsUnreachableBest(t *testing.T) {
	t.Parallel()

	best := InterfaceScore{Name: "lan0", Score: 10, Reachability: 0}
	current := InterfaceScore{Name: "wwan0", Score: 5, Reachability: 1}

	decision := DecideSwitch(best, current, true)
	if decision.ShouldSwitch {
		t.Fatalf("ShouldSwitch = true, want false")
	}
}

func TestDecideSwitchWhenCurrentIsOutsideMonitorSet(t *testing.T) {
	t.Parallel()

	best := InterfaceScore{Name: "lan0", Score: 55, Reachability: 1}
	current := InterfaceScore{Name: "eth9", Score: 0}

	decision := DecideSwitch(best, current, false)
	if !decision.ShouldSwitch {
		t.Fatalf("ShouldSwitch = false, want true; reason=%s", decision.Reason)
	}
}
