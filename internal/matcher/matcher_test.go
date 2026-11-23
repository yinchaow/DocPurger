package matcher

import (
	"testing"

	"docpurger/internal/model"
)

func TestMatcherAndOrExclude(t *testing.T) {
	// 1. AND across filename + content
	eng := New(model.SearchOptions{
		Keywords:        []string{"预算", "2026"},
		ExcludeKeywords: []string{"草稿"},
		Mode:            model.MatchModeAnd,
		Scope:           model.SearchScopeBoth,
	})

	out := eng.Evaluate("2026年度财务计划.docx", "本年度预算总额为500万元，请各部门审批。")
	if !out.Matched || !out.MatchInFilename || !out.MatchInContent {
		t.Fatalf("expected cross-scope AND match, got %+v", out)
	}
	if len(out.Snippets) < 2 {
		t.Fatalf("expected filename + content snippets, got %d", len(out.Snippets))
	}

	// Excluded keyword rejects match
	rej := eng.Evaluate("2026年度财务计划(草稿).docx", "本年度预算总额为500万元。")
	if rej.Matched {
		t.Fatalf("expected excluded keyword to reject match")
	}

	// 2. OR mode
	orEng := New(model.SearchOptions{
		Keywords: []string{"绝密", "机密"},
		Mode:     model.MatchModeOr,
		Scope:    model.SearchScopeBoth,
	})
	outOr := orEng.Evaluate("普通通知.txt", "此文件包含机密项目清单，请注意保管。")
	if !outOr.Matched || len(outOr.MatchedKeywords) != 1 || outOr.MatchedKeywords[0] != "机密" {
		t.Fatalf("unexpected OR outcome: %+v", outOr)
	}
	if len(outOr.Snippets) == 0 || len(outOr.Snippets[0].Highlights) == 0 {
		t.Fatalf("expected highlighted snippet")
	}
	snip := outOr.Snippets[0]
	runes := []rune(snip.Text)
	h := snip.Highlights[0]
	if got := string(runes[h.Start:h.End]); got != "机密" {
		t.Fatalf("highlight slice mismatch: got %q, want 机密", got)
	}
}
