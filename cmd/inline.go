package cmd

import (
	"fmt"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// fetchInlineAttachments fetches metadata for attachment IDs referenced inline in
// comments. Attachments that cannot be fetched (e.g. deleted) are skipped, so the
// returned map may have fewer entries than ids.
func fetchInlineAttachments(ids []int) map[int]*api.Attachment {
	result := make(map[int]*api.Attachment, len(ids))
	for _, id := range ids {
		var att api.Attachment
		if err := client.Get(fmt.Sprintf("/attachments/%d", id), &att); err != nil {
			continue
		}
		result[id] = &att
	}
	return result
}

// inlineAttachmentNames maps attachment ID to filename for comment rendering.
func inlineAttachmentNames(atts map[int]*api.Attachment) map[int]string {
	names := make(map[int]string, len(atts))
	for id, att := range atts {
		names[id] = att.FileName
	}
	return names
}
