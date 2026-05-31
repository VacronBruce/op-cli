package check

import "fmt"

// Level represents the severity of a check result.
type Level int

const (
	Pass Level = iota
	Warn
	Fail
)

// String returns the human-readable label for a Level.
func (l Level) String() string {
	switch l {
	case Pass:
		return "PASS"
	case Warn:
		return "WARN"
	case Fail:
		return "FAIL"
	default:
		return "UNKNOWN"
	}
}

// Result represents the outcome of a single check.
type Result struct {
	Name    string
	Level   Level
	Message string
}

// Report is the output of running all checks on a work package.
type Report struct {
	WPID    int
	Subject string
	Type    string
	Results []Result
}

// Passed returns the number of checks that passed.
func (r *Report) Passed() int {
	n := 0
	for _, res := range r.Results {
		if res.Level == Pass {
			n++
		}
	}
	return n
}

// Total returns the total number of checks run.
func (r *Report) Total() int {
	return len(r.Results)
}

// Score returns a human-readable score like "5/8".
func (r *Report) Score() string {
	return fmt.Sprintf("%d/%d", r.Passed(), r.Total())
}

// HasFailures returns true if any check resulted in Fail.
func (r *Report) HasFailures() bool {
	for _, res := range r.Results {
		if res.Level == Fail {
			return true
		}
	}
	return false
}
