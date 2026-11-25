package fileops

import (
	"os"
	"path/filepath"
	"testing"

	"docpurger/internal/model"
)

func TestSafetyGuardsQuarantineAndProtectedPaths(t *testing.T) {
	root := t.TempDir()
	subQuarantine := filepath.Join(root, "backup")
	sample := filepath.Join(root, "doc.txt")
	_ = os.WriteFile(sample, []byte("hello"), 0o644)

	// 1. Quarantine dir inside root must be rejected
	resp := DeleteFiles(model.DeleteRequest{
		Paths:         []string{sample},
		Mode:          model.DeleteModeQuarantine,
		QuarantineDir: subQuarantine,
		RootDirs:      []string{root},
	})
	if resp.Failed != 1 || resp.Succeeded != 0 {
		t.Fatalf("expected quarantine inside root to be blocked, got: %+v", resp)
	}
	if _, err := os.Stat(sample); err != nil {
		t.Fatalf("sample file should remain untouched when quarantine is blocked")
	}

	// 2. Protected paths check
	if !isProtectedSystemPath("/") {
		t.Fatalf("expected filesystem root to be protected")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if !isProtectedSystemPath(home) {
			t.Fatalf("expected user home dir %q to be protected", home)
		}
	}
}
