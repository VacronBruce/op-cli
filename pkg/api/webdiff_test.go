package api

import (
	"os"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseTextDiff_ReconstructsBeforeAndAfter(t *testing.T) {
	segs, err := ParseTextDiff(fixture(t, "diff_first.html"))
	if err != nil {
		t.Fatal(err)
	}
	wantBefore := "Overview of the <widget>\n\nSay \"hello\" to the team\n- item one\n- item two"
	wantAfter := "Draft:\n\nOverview of the <widget>\n\nSay \"hi\" to the team\n- item one\n- item two\n- item three"
	if got := Before(segs); got != wantBefore {
		t.Errorf("before = %q, want %q", got, wantBefore)
	}
	if got := After(segs); got != wantAfter {
		t.Errorf("after = %q, want %q", got, wantAfter)
	}
}

func TestParseTextDiff_ConsecutiveJournalsChain(t *testing.T) {
	first, err := ParseTextDiff(fixture(t, "diff_first.html"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := ParseTextDiff(fixture(t, "diff_second.html"))
	if err != nil {
		t.Fatal(err)
	}
	if After(first) != Before(second) {
		t.Errorf("after(first) = %q, before(second) = %q", After(first), Before(second))
	}
	if got, want := After(second), "Final:\n\nOverview of the <widget>\n\nSay \"hi\" to the team\n- item one\n- item two\n- item three (done)"; got != want {
		t.Errorf("after(second) = %q, want %q", got, want)
	}
}

func TestParseTextDiff_RejectsPagesWithoutDiff(t *testing.T) {
	if _, err := ParseTextDiff(fixture(t, "login_page.html")); err == nil {
		t.Fatal("expected an error for a page with no text-diff")
	}
}

func TestParseTextDiff_MarksChangedSegments(t *testing.T) {
	segs, _ := ParseTextDiff(fixture(t, "diff_second.html"))
	var kinds []DiffKind
	for _, s := range segs {
		if s.Kind != Same {
			kinds = append(kinds, s.Kind)
		}
	}
	want := []DiffKind{Deleted, Inserted, Inserted}
	if len(kinds) != len(want) {
		t.Fatalf("kinds = %v", kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Errorf("kinds = %v, want %v", kinds, want)
		}
	}
}

func TestDescriptionJournals_OldestFirstWithJournalIDFromPath(t *testing.T) {
	mk := func(actID int, created, raw string) Activity {
		a := Activity{ID: actID, CreatedAt: created, Details: []Formattable{{Raw: "Status changed"}, {Raw: raw}}}
		return a
	}
	ac := &ActivityCollection{}
	ac.Embedded.Elements = []Activity{
		mk(9, "2026-11-03T10:00:00Z", "Description changed (/journals/700/diff/description)"),
		{ID: 5, CreatedAt: "2026-11-02T10:00:00Z", Comment: &Formattable{Raw: "just a comment"}},
		mk(3, "2026-11-01T10:00:00Z", "Description set (/journals/655/diff/description)"),
	}
	got := DescriptionJournals(ac)
	if len(got) != 2 || got[0].JournalID != 655 || got[1].JournalID != 700 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Activity.ID != 3 {
		t.Errorf("activity not kept: %+v", got[0])
	}
}
