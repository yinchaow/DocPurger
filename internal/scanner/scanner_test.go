package scanner

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"docpurger/internal/fileops"
	"docpurger/internal/model"
)

func TestEndToEndMultiDirScanAndQuarantine(t *testing.T) {
	dirA := t.TempDir()
	dirASub := filepath.Join(dirA, "子目录")
	_ = os.MkdirAll(dirASub, 0o755)
	dirB := t.TempDir()
	quarantine := t.TempDir()

	// 1. Plain text in dirA matching AND
	p1 := filepath.Join(dirASub, "预算说明.txt")
	_ = os.WriteFile(p1, []byte("2026年信息化专项预算方案详情"), 0o644)

	// 2. Plain text matching only one keyword (should not match AND)
	p2 := filepath.Join(dirA, "普通记录.txt")
	_ = os.WriteFile(p2, []byte("2026年日常值班表"), 0o644)

	// 3. Zip archive in dirB containing matching document
	zipPath := filepath.Join(dirB, "归档.zip")
	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(zf)
	w, _ := zw.Create("内部/2026预算汇总.txt")
	_, _ = w.Write([]byte("包含2026预算及合同明细"))
	_ = zw.Close()
	_ = zf.Close()

	var mu sync.Mutex
	var matches []model.SearchResult

	// Pass dirA, dirASub (redundant child of dirA), and dirB to verify parent-child dedup + multi-root search
	summary := Run(context.Background(), model.SearchOptions{
		RootDirs:        []string{dirA, dirASub, dirB},
		Keywords:        []string{"2026", "预算"},
		Mode:            model.MatchModeAnd,
		Scope:           model.SearchScopeBoth,
		IncludeArchives: true,
	}, Callbacks{
		OnMatch: func(res model.SearchResult) {
			mu.Lock()
			matches = append(matches, res)
			mu.Unlock()
		},
	})

	if summary.Error != "" {
		t.Fatalf("unexpected scan error: %s", summary.Error)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 deduplicated matches across 2 roots, got %d: %+v", len(matches), matches)
	}
	for _, m := range matches {
		if m.RootLabel == "" {
			t.Fatalf("expected non-empty RootLabel for multi-root match: %+v", m)
		}
	}

	// Test quarantine move with multi-root dirs
	resp := fileops.DeleteFiles(model.DeleteRequest{
		Paths:         []string{p1, p1, zipPath},
		Mode:          model.DeleteModeQuarantine,
		QuarantineDir: quarantine,
		RootDirs:      []string{dirA, dirB},
		Keywords:      []string{"2026", "预算"},
	})
	if resp.Total != 2 || resp.Succeeded != 2 || resp.Failed != 0 {
		t.Fatalf("unexpected quarantine response: %+v", resp)
	}
	if _, err := os.Stat(p1); !os.IsNotExist(err) {
		t.Fatalf("expected original file to be moved")
	}
}

func TestCategoryFilterExcludesArchives(t *testing.T) {
	dir := t.TempDir()

	// Create a .zip containing a matching .docx-like text file (.rtf is in "word" category)
	zipPath := filepath.Join(dir, "资料集合.zip")
	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(zf)
	w, _ := zw.Create("预算说明.rtf")
	_, _ = w.Write([]byte(`{\rtf1 2026预算方案}`))
	_ = zw.Close()
	_ = zf.Close()

	// Also create a standalone .rtf in dir
	wordPath := filepath.Join(dir, "正式方案.rtf")
	_ = os.WriteFile(wordPath, []byte(`{\rtf1 2026预算方案}`), 0o644)

	var matches []model.SearchResult
	summary := Run(context.Background(), model.SearchOptions{
		RootDirs:        []string{dir},
		Keywords:        []string{"预算"},
		Mode:            model.MatchModeOr,
		Scope:           model.SearchScopeBoth,
		IncludeArchives: true,
		Categories:      []string{"word"},
	}, Callbacks{
		OnMatch: func(res model.SearchResult) {
			matches = append(matches, res)
		},
	})
	if summary.Error != "" {
		t.Fatalf("unexpected scan error: %s", summary.Error)
	}
	if len(matches) != 1 || matches[0].IsArchiveEntry || matches[0].FileName != "正式方案.rtf" {
		t.Fatalf("expected only standalone word file when archive category is unselected, got: %+v", matches)
	}
}
