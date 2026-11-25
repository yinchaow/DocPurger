package matcher

import (
	"sort"
	"strings"
	"unicode"

	"docpurger/internal/model"
)

// Engine performs multi-keyword AND/OR/Exclude matching and extracts highlighted snippets.
type Engine struct {
	keywords      []string
	normKeywords  []string
	normKwRunes   [][]rune
	excludes      []string
	normExcludes  []string
	mode          model.MatchMode
	scope         model.SearchScope
	caseSensitive bool
	maxSnippets   int
	snippetRadius int
}

// Outcome represents the result of evaluating a document.
type Outcome struct {
	Matched         bool
	MatchInFilename bool
	MatchInContent  bool
	MatchedKeywords []string
	TotalHits       int
	Snippets        []model.MatchSnippet
}

// New creates a matcher engine from SearchOptions.
func New(opts model.SearchOptions) *Engine {
	clean := func(list []string) ([]string, []string) {
		seen := make(map[string]struct{}, len(list))
		orig := make([]string, 0, len(list))
		norm := make([]string, 0, len(list))
		for _, raw := range list {
			kw := strings.TrimSpace(raw)
			if kw == "" {
				continue
			}
			n := kw
			if !opts.CaseSensitive {
				n = strings.ToLower(kw)
			}
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}
			orig = append(orig, kw)
			norm = append(norm, n)
		}
		return orig, norm
	}

	kws, normKws := clean(opts.Keywords)
	excs, normExcs := clean(opts.ExcludeKeywords)

	normKwRunes := make([][]rune, len(kws))
	for i, origKw := range kws {
		r := []rune(origKw)
		if !opts.CaseSensitive {
			for j := range r {
				r[j] = unicode.ToLower(r[j])
			}
		}
		normKwRunes[i] = r
	}

	mode := opts.Mode
	if mode != model.MatchModeOr {
		mode = model.MatchModeAnd
	}
	scope := opts.Scope
	if scope != model.SearchScopeFilename && scope != model.SearchScopeContent {
		scope = model.SearchScopeBoth
	}

	return &Engine{
		keywords:      kws,
		normKeywords:  normKws,
		normKwRunes:   normKwRunes,
		excludes:      excs,
		normExcludes:  normExcs,
		mode:          mode,
		scope:         scope,
		caseSensitive: opts.CaseSensitive,
		maxSnippets:   6,
		snippetRadius: 55,
	}
}

// HasKeywords returns whether at least one valid keyword is configured.
func (e *Engine) HasKeywords() bool {
	return len(e.keywords) > 0
}

// Scope returns the active search scope.
func (e *Engine) Scope() model.SearchScope {
	return e.scope
}

// Normalize lowercases text when case-insensitive matching is active.
func (e *Engine) normalize(s string) string {
	if e.caseSensitive {
		return s
	}
	return strings.ToLower(s)
}

// Evaluate checks filename and content text against keywords and exclude words.
func (e *Engine) Evaluate(fileName string, content string) Outcome {
	if len(e.keywords) == 0 {
		return Outcome{}
	}

	checkFilename := e.scope == model.SearchScopeBoth || e.scope == model.SearchScopeFilename
	checkContent := e.scope == model.SearchScopeBoth || e.scope == model.SearchScopeContent

	var normName, normContent string
	if checkFilename && fileName != "" {
		normName = e.normalize(fileName)
	}
	if checkContent && content != "" {
		normContent = e.normalize(content)
	}

	// 1. Exclude keywords take priority: if any excluded word appears in active scope, reject.
	for _, ex := range e.normExcludes {
		if checkFilename && normName != "" && strings.Contains(normName, ex) {
			return Outcome{}
		}
		if checkContent && normContent != "" && strings.Contains(normContent, ex) {
			return Outcome{}
		}
	}

	// 2. Match keywords
	matchedKws := make([]string, 0, len(e.keywords))
	var matchInName, matchInContent bool

	for i, nkw := range e.normKeywords {
		inName := checkFilename && normName != "" && strings.Contains(normName, nkw)
		inContent := checkContent && normContent != "" && strings.Contains(normContent, nkw)

		if inName {
			matchInName = true
		}
		if inContent {
			matchInContent = true
		}
		if inName || inContent {
			matchedKws = append(matchedKws, e.keywords[i])
		} else if e.mode == model.MatchModeAnd {
			return Outcome{}
		}
	}

	if len(matchedKws) == 0 {
		return Outcome{}
	}

	// 3. Count hits & build highlighted snippets
	totalHits := 0
	snippets := make([]model.MatchSnippet, 0, e.maxSnippets+1)

	if matchInName {
		nameSnip, nameHits := e.buildFilenameSnippet(fileName)
		totalHits += nameHits
		if nameSnip != nil {
			snippets = append(snippets, *nameSnip)
		}
	}

	if matchInContent {
		contentSnips, contentHits := e.buildContentSnippets(content)
		totalHits += contentHits
		snippets = append(snippets, contentSnips...)
	}

	return Outcome{
		Matched:         true,
		MatchInFilename: matchInName,
		MatchInContent:  matchInContent,
		MatchedKeywords: matchedKws,
		TotalHits:       totalHits,
		Snippets:        snippets,
	}
}

// QuickFilenameReject checks if a filename alone triggers an exclude word (fast pre-filter).
func (e *Engine) QuickFilenameReject(fileName string) bool {
	if len(e.normExcludes) == 0 || e.scope == model.SearchScopeContent {
		return false
	}
	normName := e.normalize(fileName)
	for _, ex := range e.normExcludes {
		if strings.Contains(normName, ex) {
			return true
		}
	}
	return false
}

type rawHit struct {
	start   int // rune index
	end     int // rune index
	keyword string
}

func (e *Engine) findRuneHits(text string, limit int) ([]rawHit, int) {
	origRunes := []rune(text)
	if len(origRunes) == 0 {
		return nil, 0
	}

	// Build normalized rune slice keeping 1:1 rune correspondence for safe highlighting.
	normRunes := make([]rune, len(origRunes))
	if e.caseSensitive {
		copy(normRunes, origRunes)
	} else {
		for i, r := range origRunes {
			normRunes[i] = unicode.ToLower(r)
		}
	}

	var hits []rawHit
	totalHits := 0

	for ki, kwRunes := range e.normKwRunes {
		kwLen := len(kwRunes)
		if kwLen == 0 || kwLen > len(normRunes) {
			continue
		}

		// Simple linear scan over runes
		for pos := 0; pos <= len(normRunes)-kwLen; {
			if normRunes[pos] == kwRunes[0] && runesEqual(normRunes[pos:pos+kwLen], kwRunes) {
				totalHits++
				if len(hits) < limit {
					hits = append(hits, rawHit{
						start:   pos,
						end:     pos + kwLen,
						keyword: e.keywords[ki],
					})
				}
				pos += kwLen
			} else {
				pos++
			}
		}
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].start != hits[j].start {
			return hits[i].start < hits[j].start
		}
		return hits[i].end > hits[j].end
	})

	return hits, totalHits
}

func runesEqual(a, b []rune) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (e *Engine) buildFilenameSnippet(fileName string) (*model.MatchSnippet, int) {
	hits, total := e.findRuneHits(fileName, 20)
	if len(hits) == 0 {
		return nil, 0
	}
	spans := mergeSpans(hits, 0)
	return &model.MatchSnippet{
		Source:     "filename",
		Text:       fileName,
		Highlights: spans,
	}, total
}

func (e *Engine) buildContentSnippets(content string) ([]model.MatchSnippet, int) {
	// Normalize whitespace lightly for display while keeping rune indices consistent
	cleaned := compactWhitespace(content)
	origRunes := []rune(cleaned)
	hits, total := e.findRuneHits(cleaned, 120)
	if len(hits) == 0 {
		return nil, 0
	}

	// Pick diverse hits: prioritize covering every distinct matched keyword first
	selected := make([]rawHit, 0, e.maxSnippets)
	seenKw := make(map[string]bool)
	for _, h := range hits {
		if !seenKw[h.keyword] {
			seenKw[h.keyword] = true
			selected = append(selected, h)
			if len(selected) >= e.maxSnippets {
				break
			}
		}
	}
	for _, h := range hits {
		if len(selected) >= e.maxSnippets {
			break
		}
		// Skip hits too close to already selected windows
		tooClose := false
		for _, s := range selected {
			if absInt(h.start-s.start) < e.snippetRadius {
				tooClose = true
				break
			}
		}
		if !tooClose {
			selected = append(selected, h)
		}
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].start < selected[j].start
	})

	snippets := make([]model.MatchSnippet, 0, len(selected))
	for _, center := range selected {
		winStart := center.start - e.snippetRadius
		if winStart < 0 {
			winStart = 0
		}
		winEnd := center.end + e.snippetRadius
		if winEnd > len(origRunes) {
			winEnd = len(origRunes)
		}

		prefix := ""
		prefixLen := 0
		if winStart > 0 {
			prefix = "…"
			prefixLen = 1
		}
		suffix := ""
		if winEnd < len(origRunes) {
			suffix = "…"
		}

		// Collect all hits falling within [winStart, winEnd]
		var winHits []rawHit
		for _, h := range hits {
			if h.start >= winStart && h.end <= winEnd {
				winHits = append(winHits, rawHit{
					start:   h.start - winStart + prefixLen,
					end:     h.end - winStart + prefixLen,
					keyword: h.keyword,
				})
			}
		}

		snippetText := prefix + string(origRunes[winStart:winEnd]) + suffix
		snippets = append(snippets, model.MatchSnippet{
			Source:     "content",
			Text:       snippetText,
			Highlights: mergeSpans(winHits, 0),
		})
	}

	return snippets, total
}

func mergeSpans(hits []rawHit, offset int) []model.HighlightSpan {
	if len(hits) == 0 {
		return nil
	}
	spans := make([]model.HighlightSpan, 0, len(hits))
	for _, h := range hits {
		s := h.start + offset
		e := h.end + offset
		if len(spans) > 0 && s <= spans[len(spans)-1].End {
			if e > spans[len(spans)-1].End {
				spans[len(spans)-1].End = e
			}
			continue
		}
		spans = append(spans, model.HighlightSpan{
			Start:   s,
			End:     e,
			Keyword: h.keyword,
		})
	}
	return spans
}

func compactWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	lastSpace := false
	for _, r := range s {
		if r == '\r' || r == '\n' || r == '\t' || r == ' ' || r == '\u00a0' || r == '\u3000' {
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
		lastSpace = false
	}
	return strings.TrimSpace(b.String())
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
