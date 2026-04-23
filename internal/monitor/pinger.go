package monitor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type PingResult struct {
	Target       string
	Success      bool
	PacketLoss   float64
	AverageRTTMS float64
	RawOutput    string
	Error        error
}

type Pinger struct {
	Timeout time.Duration
	Count   int
}

var (
	lossRegexp = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)% packet loss`)
	rttRegexp  = regexp.MustCompile(`= [0-9.]+/[0-9.]+/([0-9.]+)/[0-9.]+ ms`)
)

func (p Pinger) Ping(ctx context.Context, iface, target string) PingResult {
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	args := []string{"-I", iface, "-c", strconv.Itoa(p.Count), "-W", strconv.Itoa(int(p.Timeout.Seconds())), target}
	cmd := exec.CommandContext(ctx, "ping", args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
	if output == "" {
		output = strings.TrimSpace(stderr.String())
	}

	result := PingResult{Target: target, RawOutput: output}
	if err != nil {
		result.Error = fmt.Errorf("ping target %s via %s: %w", target, iface, err)
	}

	if match := lossRegexp.FindStringSubmatch(output); len(match) == 2 {
		loss, parseErr := strconv.ParseFloat(match[1], 64)
		if parseErr == nil {
			result.PacketLoss = loss
		}
	}

	if match := rttRegexp.FindStringSubmatch(output); len(match) == 2 {
		avg, parseErr := strconv.ParseFloat(match[1], 64)
		if parseErr == nil {
			result.AverageRTTMS = avg
		}
	}

	result.Success = result.PacketLoss < 100
	return result
}
