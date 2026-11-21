package check

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/chenhuijun/op-cli/pkg/api"
)

var defaultDesignHosts = []string{"figma.com"}

var readyStatuses = []string{"done", "closed", "completed", "resolved", "ready for release"}

// Runner executes checks against work packages.
type Runner struct {
	Client api.APIClient
	// Config is the Definition of Ready to apply. When nil, the baked-in
	// default (defaultDoR) is used.
	Config *DoRConfig
	// DesignHosts defaults to figma.com when empty.
	DesignHosts []string
}

// Run fetches a work package and runs type-appropriate checks against it.
func (r *Runner) Run(id int) (*Report, error) {
	wp, err := r.Client.GetWorkPackage(id)
	if err != nil {
		return nil, fmt.Errorf("getting work package %d: %w", id, err)
	}

	attachmentCount, hasVisual := 0, false
	if att, err := r.Client.ListAttachments(id); err == nil {
		attachmentCount = att.Total
		hasVisual = hasVisualAttachment(att)
	}

	// Screenshots are often pasted inline into comments. Those live in
	// Activity::Comment containers, so they are absent from the /attachments
	// endpoint; count them too so a ticket whose only evidence is in a
	// comment is not falsely flagged as having no attachments.
	if ac, err := r.Client.ListActivities(id); err == nil {
		attachmentCount += len(api.CommentInlineAttachmentIDs(ac))
	}

	// Mockups are often attached to the parent.
	parentAttachments, parentDescription := 0, ""
	if parentID, ok := wpIDFromHref(wp.Links.Parent.Href); ok {
		if att, err := r.Client.ListAttachments(parentID); err == nil {
			parentAttachments = att.Total
			hasVisual = hasVisual || hasVisualAttachment(att)
		}
		if parent, err := r.Client.GetWorkPackage(parentID); err == nil && parent.Description != nil {
			parentDescription = parent.Description.Raw
		}
	}

	// INVEST "Independent": count the work packages this ticket is blocked by,
	// and find related upstream tickets that are not done. Non-fatal — a
	// relations fetch failure leaves both empty rather than failing the run.
	blockedBy := 0
	var openUpstream []Upstream
	if rc, err := r.Client.ListRelations(id); err == nil {
		blockedBy = blockedByCount(id, rc)
		openUpstream = r.openUpstream(id, rc)
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

	designHosts := r.DesignHosts
	if len(designHosts) == 0 {
		designHosts = defaultDesignHosts
	}
	in := CheckInput{
		AttachmentCount:       attachmentCount,
		BlockedByCount:        blockedBy,
		ParentAttachmentCount: parentAttachments,
		HasVisualAttachment:   hasVisual,
		ParentDescription:     parentDescription,
		DesignHosts:           designHosts,
		OpenUpstream:          openUpstream,
	}
	for _, rule := range checks {
		report.Results = append(report.Results, rule(wp, in))
	}

	return report, nil
}

func hasVisualAttachment(ac *api.AttachmentCollection) bool {
	for _, a := range ac.Embedded.Elements {
		ct := strings.ToLower(a.ContentType)
		if strings.HasPrefix(ct, "image/") || strings.HasPrefix(ct, "video/") {
			return true
		}
	}
	return false
}

func wpIDFromHref(href string) (int, bool) {
	i := strings.LastIndex(href, "/work_packages/")
	if i < 0 {
		return 0, false
	}
	id, err := strconv.Atoi(href[i+len("/work_packages/"):])
	return id, err == nil
}

var upstreamSubjectRe = regexp.MustCompile(`(?i)\b(api|apis|backend|server|web|wp|wordpress)\b`)

// openUpstream returns the work packages related to wpID (type "relates") whose
// subject looks upstream and whose status is not a ready one, fetched with a
// single filtered list call. Any fetch failure yields none.
func (r *Runner) openUpstream(wpID int, rc *api.RelationCollection) []Upstream {
	var ids []string
	for _, rel := range rc.Embedded.Elements {
		if !strings.EqualFold(rel.Type, "relates") {
			continue
		}
		for _, href := range []string{rel.Links.From.Href, rel.Links.To.Href} {
			if id, ok := wpIDFromHref(href); ok && id != wpID {
				ids = append(ids, strconv.Itoa(id))
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	list, err := r.Client.ListAllWorkPackages([]api.Filter{api.NewFilter("id", "=", ids...)}, "", len(ids))
	if err != nil {
		return nil
	}
	var open []Upstream
	for _, rel := range list.Embedded.Elements {
		status := rel.Links.Status.Title
		if upstreamSubjectRe.MatchString(rel.Subject) && !slices.Contains(readyStatuses, strings.ToLower(status)) {
			open = append(open, Upstream{ID: rel.ID, Status: status})
		}
	}
	slices.SortFunc(open, func(a, b Upstream) int { return a.ID - b.ID })
	return open
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
