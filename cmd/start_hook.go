package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/chenhuijun/op-cli/pkg/check"
	"github.com/spf13/viper"
)

var startHookTimeout = 30 * time.Second

type startEvent struct {
	Event      string           `json:"event"`
	At         string           `json:"at"`
	ID         int              `json:"id"`
	Subject    string           `json:"subject"`
	Type       string           `json:"type"`
	Project    string           `json:"project"`
	Branch     string           `json:"branch"`
	Check      *startEventCheck `json:"check"`
	CheckError string           `json:"check_error,omitempty"`
}

type startEventCheck struct {
	Readiness string             `json:"readiness"`
	Score     int                `json:"score"`
	Total     int                `json:"total"`
	Percent   int                `json:"percent"`
	Results   []startEventResult `json:"results"`
}

type startEventResult struct {
	Name    string `json:"name"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

func newStartEvent(id int, subject, typeName, project, branch string, report *check.Report, checkErr error) startEvent {
	ev := startEvent{
		Event:   "ticket_start",
		At:      time.Now().Format(time.RFC3339),
		ID:      id,
		Subject: subject,
		Type:    typeName,
		Project: project,
		Branch:  branch,
	}
	if checkErr != nil {
		ev.CheckError = checkErr.Error()
		return ev
	}
	ev.Check = &startEventCheck{
		Readiness: report.Readiness(),
		Score:     report.Passed(),
		Total:     report.Total(),
		Percent:   report.ScorePercent(),
		Results:   make([]startEventResult, 0, len(report.Results)),
	}
	for _, r := range report.Results {
		ev.Check.Results = append(ev.Check.Results, startEventResult{r.Name, r.Level.String(), r.Message})
	}
	return ev
}

func runStartHook(ev startEvent) {
	hook := viper.GetString("start_hook")
	if hook == "" {
		return
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: start_hook skipped: %v\n", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), startHookTimeout)
	defer cancel()
	c := exec.CommandContext(ctx, "sh", "-c", hook)
	c.Stdin = bytes.NewReader(payload)
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	c.WaitDelay = time.Second
	if err := c.Run(); err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("timed out after %s", startHookTimeout)
		}
		fmt.Fprintf(os.Stderr, "warning: start_hook failed: %v\n", err)
	}
}
