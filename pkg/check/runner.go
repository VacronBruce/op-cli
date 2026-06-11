package check

import (
	"fmt"
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

	typeName := wp.Links.Type.Title
	checks := RulesForType(typeName)

	report := &Report{
		WPID:    wp.ID,
		Subject: wp.Subject,
		Type:    typeName,
	}

	for _, rule := range checks {
		report.Results = append(report.Results, rule(wp, attachmentCount))
	}

	return report, nil
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
