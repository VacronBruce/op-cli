package check

import (
	"fmt"
	"strings"
	"sync"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// attachmentCollection mirrors the response from the attachments endpoint.
type attachmentCollection struct {
	Total int `json:"total"`
}

// Runner executes checks against work packages.
type Runner struct {
	Client api.APIClient
	// Config is the Definition of Ready to apply. When nil, the baked-in
	// default (defaultDoR) is used.
	Config *DoRConfig
}

// Run fetches a work package and runs type-appropriate checks against it.
func (r *Runner) Run(id int) (*Report, error) {
	wp, err := r.Client.GetWorkPackage(id)
	if err != nil {
		return nil, fmt.Errorf("getting work package %d: %w", id, err)
	}

	var att attachmentCollection
	if err := r.Client.Get(fmt.Sprintf("/work_packages/%d/attachments", id), &att); err != nil {
		// Non-fatal: continue with zero attachments
		att.Total = 0
	}

	// Screenshots are often pasted inline into comments. Those live in
	// Activity::Comment containers, so they are absent from the /attachments
	// endpoint above; count them too so a ticket whose only evidence is in a
	// comment is not falsely flagged as having no attachments.
	attachmentCount := att.Total
	if ac, err := r.Client.ListActivities(id); err == nil {
		attachmentCount += len(api.CommentInlineAttachmentIDs(ac))
	}

	// INVEST "Independent": count the work packages this ticket is blocked by.
	// Non-fatal — a relations fetch failure leaves the count at zero rather than
	// failing the whole check run.
	blockedBy := 0
	if rc, err := r.Client.ListRelations(id); err == nil {
		blockedBy = blockedByCount(id, rc)
	}

	typeName := wp.Links.Type.Title
	cfg := r.Config
	if cfg == nil {
		cfg = defaultDoR
	}
	checks := cfg.Rules(typeName)

	report := &Report{
		WPID:    wp.ID,
		Subject: wp.Subject,
		Type:    typeName,
	}

	in := CheckInput{AttachmentCount: attachmentCount, BlockedByCount: blockedBy}
	for _, rule := range checks {
		report.Results = append(report.Results, rule(wp, in))
	}

	return report, nil
}

// blockedByCount reports how many relations make wpID depend on another work
// package: it is the "to" end of a "blocks" relation (something blocks it) or the
// "from" end of a "blocked" relation. This is the INVEST "Independent" signal.
func blockedByCount(wpID int, rc *api.RelationCollection) int {
	if rc == nil {
		return 0
	}
	n := 0
	for _, rel := range rc.Embedded.Elements {
		switch strings.ToLower(rel.Type) {
		case "blocks":
			if hrefIsWP(rel.Links.To.Href, wpID) {
				n++
			}
		case "blocked":
			if hrefIsWP(rel.Links.From.Href, wpID) {
				n++
			}
		}
	}
	return n
}

// hrefIsWP reports whether an API href points at the given work-package id.
func hrefIsWP(href string, wpID int) bool {
	return strings.HasSuffix(href, fmt.Sprintf("/work_packages/%d", wpID))
}

// batchConcurrency bounds parallel checks so a 40-ticket sprint doesn't open
// 120 simultaneous connections against the OpenProject server.
const batchConcurrency = 5

// RunBatch runs checks on a slice of work packages. Each check is several
// API round-trips, so checks run concurrently (bounded); reports stay in
// input order and any failure fails the batch naming the first failing
// ticket in input order, same as the sequential contract.
func (r *Runner) RunBatch(wps []api.WorkPackage) ([]Report, error) {
	reports := make([]Report, len(wps))
	errs := make([]error, len(wps))
	sem := make(chan struct{}, batchConcurrency)
	var wg sync.WaitGroup
	for i, wp := range wps {
		wg.Add(1)
		go func(i, id int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			report, err := r.Run(id)
			if err != nil {
				errs[i] = fmt.Errorf("checking #%d: %w", id, err)
				return
			}
			reports[i] = *report
		}(i, wp.ID)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return reports, nil
}
