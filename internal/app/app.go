package app

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"netmonitor/internal/config"
	"netmonitor/internal/evaluator"
	"netmonitor/internal/monitor"
	"netmonitor/internal/network"
	"netmonitor/internal/system"
)

func Run(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	log.Printf("loaded %d configured interfaces and %d ping targets", len(cfg.Interfaces), len(cfg.PingTargets))

	pinger := monitor.Pinger{
		Timeout: time.Duration(cfg.PingTimeoutSec) * time.Second,
		Count:   cfg.PingCount,
	}
	eval := evaluator.Evaluator{}
	route := network.RouteSwitcher{}
	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	var previousActive []string
	for {
		activeInterfaces := filterInterfaces(cfg.Interfaces)
		logInterfaceChanges(previousActive, activeInterfaces)
		previousActive = append(previousActive[:0], activeInterfaces...)

		if len(activeInterfaces) == 0 {
			log.Print("no active configured interfaces detected, skip this round and wait for next check")
		} else {
			runCheck(ctx, activeInterfaces, cfg.PingTargets, pinger, eval, route)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func runCheck(
	ctx context.Context,
	interfaces []string,
	targets []string,
	pinger monitor.Pinger,
	eval evaluator.Evaluator,
	route network.RouteSwitcher,
) {
	var scores []evaluator.InterfaceScore
	for _, iface := range interfaces {
		var samples []monitor.PingResult
		for _, target := range targets {
			result := pinger.Ping(ctx, iface, target)
			samples = append(samples, result)
			log.Printf("iface=%s target=%s %s", iface, target, formatPingResult(result))
		}

		score := eval.Evaluate(iface, samples)
		scores = append(scores, score)
		log.Printf("iface=%s summary score=%.2f reachability=%.2f loss=%.2f%% avg_rtt=%.2fms", score.Name, score.Score, score.Reachability, score.PacketLoss, score.AverageRTTMS)
	}

	if len(scores) == 0 {
		log.Print("no interface score available")
		return
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})
	log.Printf("score ranking: %s", formatScoreRanking(scores))

	// The top-ranked interface is the preferred candidate for this round.
	best := scores[0]

	currentIface, err := route.CurrentDefaultInterface(ctx)
	if err != nil {
		log.Printf("skip switching, current route unavailable: %v; best_candidate=%s score=%.2f", err, best.Name, best.Score)
		return
	}

	current, currentTracked := findScore(scores, currentIface)

	// Switching is a separate decision from scoring: even if one interface ranks
	// first, we still require the decision logic to confirm the gain is meaningful.
	decision := evaluator.DecideSwitch(best, current, currentTracked)
	if !decision.ShouldSwitch {
		if currentTracked {
			log.Printf("keep current interface=%s reason=%s current_score=%.2f best_candidate=%s best_score=%.2f", currentIface, decision.Reason, current.Score, best.Name, best.Score)
		} else {
			log.Printf("keep current interface=%s reason=%s best_candidate=%s best_score=%.2f", currentIface, decision.Reason, best.Name, best.Score)
		}
		return
	}

	if err := route.SwitchDefaultInterface(ctx, best.Name); err != nil {
		log.Printf("switch interface failed current=%s target=%s reason=%s err=%v", currentIface, best.Name, decision.Reason, err)
		return
	}

	log.Printf("switched default interface from %s to %s reason=%s", currentIface, best.Name, decision.Reason)
}

func filterInterfaces(interfaces []string) []string {
	result := make([]string, 0, len(interfaces))
	for _, iface := range interfaces {
		if !system.InterfaceExists(iface) {
			log.Printf("skip missing interface: %s", iface)
			continue
		}
		result = append(result, iface)
	}
	return result
}

func logInterfaceChanges(previous, current []string) {
	added, removed := detectInterfaceChanges(previous, current)
	if len(previous) == 0 && len(current) > 0 {
		log.Printf("active interfaces detected: %s", strings.Join(current, ", "))
		return
	}

	for _, iface := range added {
		log.Printf("interface became available: %s", iface)
	}
	for _, iface := range removed {
		log.Printf("interface became unavailable: %s", iface)
	}
}

func detectInterfaceChanges(previous, current []string) (added, removed []string) {
	prevSet := make(map[string]struct{}, len(previous))
	currSet := make(map[string]struct{}, len(current))
	for _, iface := range previous {
		prevSet[iface] = struct{}{}
	}
	for _, iface := range current {
		currSet[iface] = struct{}{}
		if _, ok := prevSet[iface]; !ok {
			added = append(added, iface)
		}
	}
	for _, iface := range previous {
		if _, ok := currSet[iface]; !ok {
			removed = append(removed, iface)
		}
	}
	return added, removed
}

func findScore(scores []evaluator.InterfaceScore, iface string) (evaluator.InterfaceScore, bool) {
	for _, score := range scores {
		if score.Name == iface {
			return score, true
		}
	}
	return evaluator.InterfaceScore{Name: iface}, false
}

func formatPingResult(result monitor.PingResult) string {
	status := "success"
	if result.Error != nil {
		status = fmt.Sprintf("error=%v", result.Error)
	}
	return fmt.Sprintf("%s loss=%.2f%% avg_rtt=%.2fms", status, result.PacketLoss, result.AverageRTTMS)
}

func formatScoreRanking(scores []evaluator.InterfaceScore) string {
	parts := make([]string, 0, len(scores))
	for _, score := range scores {
		parts = append(parts, fmt.Sprintf("%s(%.2f)", score.Name, score.Score))
	}
	return strings.Join(parts, " > ")
}
