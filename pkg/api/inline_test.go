package api

import (
	"reflect"
	"testing"
)

func TestInlineAttachmentIDs(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []int
	}{
		{
			name: "no images",
			raw:  "plain comment with no attachments",
			want: []int{},
		},
		{
			name: "single html img tag",
			raw:  `before <img class="op-uc-image op-uc-image_inline" src="/api/v3/attachments/43221/content"> after`,
			want: []int{43221},
		},
		{
			name: "multiple images keep first-seen order",
			raw: `a <img src="/api/v3/attachments/43221/content">` +
				` b <img src="/api/v3/attachments/43222/content">` +
				` c <img src="/api/v3/attachments/43224/content">`,
			want: []int{43221, 43222, 43224},
		},
		{
			name: "duplicate references are deduped",
			raw:  `x /api/v3/attachments/43221/content y /api/v3/attachments/43221/content`,
			want: []int{43221},
		},
		{
			name: "markdown image form",
			raw:  `![shot](/api/v3/attachments/55/content)`,
			want: []int{55},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InlineAttachmentIDs(tt.raw)
			if len(tt.want) == 0 && len(got) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InlineAttachmentIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommentInlineAttachmentIDs_DedupAcrossComments(t *testing.T) {
	ac := &ActivityCollection{}
	ac.Embedded.Elements = []Activity{
		{ID: 1, Comment: &Formattable{Raw: `<img src="/api/v3/attachments/43221/content">`}},
		{ID: 2, Comment: nil}, // a journal change with no comment must be skipped
		{ID: 3, Comment: &Formattable{Raw: `<img src="/api/v3/attachments/43222/content"> /api/v3/attachments/43221/content`}},
	}

	got := CommentInlineAttachmentIDs(ac)
	want := []int{43221, 43222}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CommentInlineAttachmentIDs() = %v, want %v", got, want)
	}
}

func TestCommentInlineAttachmentIDs_Nil(t *testing.T) {
	if got := CommentInlineAttachmentIDs(nil); got != nil {
		t.Errorf("expected nil for nil collection, got %v", got)
	}
}
