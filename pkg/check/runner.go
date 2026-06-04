package check

import (
	"fmt"

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

	for _, check := range checks {
		report.Results = append(report.Results, check(wp, attachmentCount))
	}

	return report, nil
}

// RunBatch runs checks on a slice of work packages.
func (r *Runner) RunBatch(wps []api.WorkPackage) ([]Report, error) {
	var reports []Report
	for _, wp := range wps {
		report, err := r.Run(wp.ID)
		if err != nil {
			return nil, fmt.Errorf("checking #%d: %w", wp.ID, err)
		}
		reports = append(reports, *report)
	}
	return reports, nil
}
