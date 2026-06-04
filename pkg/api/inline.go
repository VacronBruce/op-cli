package api

import (
	"regexp"
	"strconv"
)

// inlineAttachmentRe matches the attachment reference embedded in a comment body,
// e.g. <img src="/api/v3/attachments/43221/content"> or the markdown equivalent
// ![](/api/v3/attachments/43221/content). Only the numeric ID is captured.
var inlineAttachmentRe = regexp.MustCompile(`/attachments/(\d+)/content`)

// InlineAttachmentIDs returns the attachment IDs referenced inline in a single
// comment body, in first-seen order with duplicates removed.
func InlineAttachmentIDs(raw string) []int {
	matches := inlineAttachmentRe.FindAllStringSubmatch(raw, -1)
	ids := make([]int, 0, len(matches))
	seen := make(map[int]bool, len(matches))
	for _, m := range matches {
		id, err := strconv.Atoi(m[1])
		if err != nil || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

// CommentInlineAttachmentIDs returns the unique attachment IDs referenced inline
// across every comment in the collection, in first-seen order. Inline images live
// in Activity::Comment containers, so they do not appear in the work package's
// /attachments endpoint and must be discovered from the comment bodies.
func CommentInlineAttachmentIDs(ac *ActivityCollection) []int {
	if ac == nil {
		return nil
	}
	var ids []int
	seen := make(map[int]bool)
	for _, a := range ac.Embedded.Elements {
		if a.Comment == nil {
			continue
		}
		for _, id := range InlineAttachmentIDs(a.Comment.Raw) {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}
