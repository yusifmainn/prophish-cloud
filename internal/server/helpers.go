package server

import (
	"strings"

	"secdrill/internal/models"
)

// parseHeaderLines turns a textarea of "Key: Value" lines into a header map.
func parseHeaderLines(s string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k != "" {
			out[k] = v
		}
	}
	return out
}

// Percentages holds click-through style rates for a campaign report.
type Percentages struct {
	Sent      int
	Opened    int
	Clicked   int
	Submitted int
	Reported  int
}

// percentages computes each stage as a percentage of total targets.
func percentages(s models.ResultStats) Percentages {
	if s.Total == 0 {
		return Percentages{}
	}
	pct := func(n int) int { return models.Pct(n, s.Total) }
	return Percentages{
		Sent:      pct(s.Sent),
		Opened:    pct(s.Opened),
		Clicked:   pct(s.Clicked),
		Submitted: pct(s.Submitted),
		Reported:  pct(s.Reported),
	}
}
