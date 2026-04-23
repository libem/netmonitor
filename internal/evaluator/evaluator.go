package evaluator

import (
	"fmt"
	"math"

	"netmonitor/internal/monitor"
)

const MinSwitchScoreDelta = 5

type InterfaceScore struct {
	Name         string
	Score        float64
	Reachability float64
	PacketLoss   float64
	AverageRTTMS float64
	Samples      []monitor.PingResult
}

type Evaluator struct{}

// Decision explains whether a route switch should happen and why.
type Decision struct {
	ShouldSwitch bool
	Reason       string
	ScoreDelta   float64
}

func (Evaluator) Evaluate(name string, samples []monitor.PingResult) InterfaceScore {
	score := InterfaceScore{Name: name, Samples: samples}
	if len(samples) == 0 {
		return score
	}

	var successCount int
	var totalLoss float64
	var totalRTT float64
	var rttSamples int

	for _, sample := range samples {
		totalLoss += sample.PacketLoss
		if sample.Success {
			successCount++
		}
		if sample.AverageRTTMS > 0 {
			totalRTT += sample.AverageRTTMS
			rttSamples++
		}
	}

	score.Reachability = float64(successCount) / float64(len(samples))
	score.PacketLoss = totalLoss / float64(len(samples))
	if rttSamples > 0 {
		score.AverageRTTMS = totalRTT / float64(rttSamples)
	} else {
		// If an interface never returns RTT, treat it as extremely slow so it
		// loses in ranking even when the command itself happened to return output.
		score.AverageRTTMS = 9999
	}

	// Scoring idea:
	// 1. Reachability is the strongest signal, so it contributes up to 100 points.
	// 2. Latency adds a capped penalty to avoid a very large RTT dominating forever.
	// 3. Packet loss adds another penalty because unstable links should lose rank.
	latencyPenalty := math.Min(score.AverageRTTMS, 1000) / 10
	lossPenalty := score.PacketLoss * 0.7
	reachabilityBonus := score.Reachability * 100

	score.Score = reachabilityBonus - latencyPenalty - lossPenalty
	return score
}

func DecideSwitch(best, current InterfaceScore, currentTracked bool) Decision {
	if best.Name == "" {
		return Decision{Reason: "no candidate interface available"}
	}

	// Never switch to a link that cannot reach any target in the current round.
	if best.Reachability == 0 {
		return Decision{Reason: fmt.Sprintf("best candidate %s has zero reachability", best.Name)}
	}

	if currentTracked && best.Name == current.Name {
		return Decision{Reason: fmt.Sprintf("current interface %s is already the best candidate", current.Name)}
	}

	if !currentTracked {
		return Decision{
			ShouldSwitch: true,
			Reason:       fmt.Sprintf("current interface is outside monitor set, switch to %s", best.Name),
			ScoreDelta:   best.Score,
		}
	}

	delta := best.Score - current.Score
	// Keep the current route when the gap is too small to avoid flapping caused
	// by small RTT/loss jitter between two similarly healthy links.
	if delta < MinSwitchScoreDelta {
		return Decision{
			Reason:     fmt.Sprintf("score delta %.2f is below switch threshold %d", delta, MinSwitchScoreDelta),
			ScoreDelta: delta,
		}
	}

	return Decision{
		ShouldSwitch: true,
		Reason:       fmt.Sprintf("%s outperforms %s by %.2f points", best.Name, current.Name, delta),
		ScoreDelta:   delta,
	}
}
