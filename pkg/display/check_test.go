package display

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/check"
)

func sampleReport() *check.Report {
	return &check.Report{
		WPID:    81321,
		Subject: "Crash on save",
		Type:    "Bug",
		Results: []check.Result{
			{Name: "Has description", Level: check.Pass},
			{Name: "Priority explicitly set", Level: check.Warn, Message: "Priority not set"},
			{Name: "Has attachments", Level: check.Fail, Message: "No attachments"},
		},
	}
}

func TestCheckReport_ShowsLevelsAndMessages(t *testing.T) {
	// The terminal report is the dev's fix-list: each rule line must carry its
	// level label and the message explaining what to fix.
	out := testutil.CaptureStdout(func() { CheckReport(sampleReport()) })

	if !strings.Contains(out, "#81321 Crash on save") {
		t.Errorf("expected header, got: %s", out)
	}
	// 1 Pass + 1 Warn + 1 Fail → (100 + 50) / 3 = 50%; a Fail blocks the gate.
	if !strings.Contains(out, "Score: 1/3 (50%) — NEEDS WORK") {
		t.Errorf("expected weighted score and DoR gate, got: %s", out)
	}
	for _, want := range []string{"PASS", "WARN", "FAIL", "(Priority not set)", "(No attachments)"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in report, got: %s", want, out)
		}
	}
}

func TestCheckSummary_CountsWarnOnlyAsFullyPassed(t *testing.T) {
	// Warnings don't block readiness: a warn-only ticket counts as fully
	// passed (strict mode promotes warns BEFORE calling this, by design).
	warnOnly := &check.Report{WPID: 1, Subject: "warny", Type: "Task", Results: []check.Result{
		{Name: "a", Level: check.Pass},
		{Name: "b", Level: check.Warn},
	}}
	failing := &check.Report{WPID: 2, Subject: "saddy", Type: "Task", Results: []check.Result{
		{Name: "a", Level: check.Fail},
		{Name: "b", Level: check.Pass},
	}}

	out := testutil.CaptureStdout(func() {
		CheckSummary([]check.Report{*warnOnly, *failing}, "Sprint 24")
	})

	if !strings.Contains(out, "Sprint Readiness: Sprint 24") {
		t.Errorf("expected sprint header, got: %s", out)
	}
	if !strings.Contains(out, "Summary: 1/2 fully pass") {
		t.Errorf("warn-only must count as fully passed, got: %s", out)
	}
}

func TestCheckSummary_Empty(t *testing.T) {
	out := testutil.CaptureStdout(func() { CheckSummary(nil, "Sprint 24") })
	if !strings.Contains(out, "No work packages to check.") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestCheckReportMarkdown_TableWithIcons(t *testing.T) {
	// The markdown rendering is what gets POSTED to the ticket — it must be a
	// valid table with consistent lowercase icons and the rule messages inline.
	md := CheckReportMarkdown(sampleReport())

	if !strings.Contains(md, "## Readiness Check — 1/3 passed (50% — NEEDS WORK)") {
		t.Errorf("expected score headline with percent and gate, got: %s", md)
	}
	if !strings.Contains(md, "| Status | Check |") {
		t.Errorf("expected table header, got: %s", md)
	}
	for _, want := range []string{"| ok | Has description |", "| warn |", "| fail |", "— No attachments"} {
		if !strings.Contains(md, want) {
			t.Errorf("expected %q in markdown, got: %s", want, md)
		}
	}
	if !strings.Contains(md, "*Run by `op check`*") {
		t.Errorf("expected footer, got: %s", md)
	}
}
