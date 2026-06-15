package api

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// hoursPerDay is the conversion used between day and hour estimates. OpenProject
// stores "Work" (estimatedTime) in hours and renders days using the instance's
// hours-per-day setting; 8 is the OpenProject default and matches the UI's
// "2d 0h" rendering.
const hoursPerDay = 8.0

// estimateToken matches one "<number><unit>" piece, e.g. "2d", "1.5h", or a
// bare number (treated as hours).
var estimateToken = regexp.MustCompile(`(?i)^([0-9]*\.?[0-9]+)\s*([dh]?)$`)

// ParseEstimate converts a human estimate ("2d", "16h", "2d 4h", "1.5h", "16")
// into an ISO 8601 duration string suitable for the estimatedTime field
// (e.g. "PT16H"). Days are converted at 8h/day. A bare number is hours.
func ParseEstimate(s string) (string, error) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return "", fmt.Errorf("empty estimate")
	}

	var totalHours float64
	for _, f := range fields {
		m := estimateToken.FindStringSubmatch(f)
		if m == nil {
			return "", fmt.Errorf("invalid estimate %q (use forms like 2d, 16h, \"2d 4h\")", s)
		}
		n, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return "", fmt.Errorf("invalid estimate %q: %w", s, err)
		}
		switch strings.ToLower(m[2]) {
		case "d":
			totalHours += n * hoursPerDay
		default: // "h" or bare number
			totalHours += n
		}
	}

	if totalHours <= 0 {
		return "", fmt.Errorf("estimate must be greater than zero")
	}

	// OpenProject serializes estimatedTime in hours; emit PT<H>H[<M>M].
	hours := int(totalHours)
	minutes := int((totalHours - float64(hours)) * 60.0)
	iso := "PT"
	if hours > 0 {
		iso += fmt.Sprintf("%dH", hours)
	}
	if minutes > 0 {
		iso += fmt.Sprintf("%dM", minutes)
	}
	return iso, nil
}

// estimateParts pulls day/hour/minute/second counts out of an ISO 8601 duration.
var estimateParts = regexp.MustCompile(`(?i)P(?:([0-9.]+)D)?(?:T(?:([0-9.]+)H)?(?:([0-9.]+)M)?(?:([0-9.]+)S)?)?`)

// FormatEstimate renders an ISO 8601 duration (as returned in estimatedTime,
// e.g. "PT16H") back as "Xd Yh", matching the OpenProject UI. Returns "" for an
// empty or unparseable value. OpenProject emits hours-only durations for
// estimates; a D component, if present, is taken as the ISO-standard 24h.
func FormatEstimate(iso string) string {
	if iso == "" {
		return ""
	}
	m := estimateParts.FindStringSubmatch(iso)
	if m == nil {
		return ""
	}
	parse := func(s string) float64 {
		if s == "" {
			return 0
		}
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}
	totalHours := parse(m[1])*24.0 + parse(m[2]) + parse(m[3])/60.0 + parse(m[4])/3600.0
	if totalHours <= 0 {
		return ""
	}

	days := int(totalHours / hoursPerDay)
	rem := totalHours - float64(days)*hoursPerDay
	if days > 0 {
		return fmt.Sprintf("%dd %gh", days, rem)
	}
	return fmt.Sprintf("%gh", rem)
}
