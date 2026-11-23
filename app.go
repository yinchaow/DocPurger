package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"docpurger/internal/fileops"
	"docpurger/internal/model"
	"docpurger/internal/scanner"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const AppVersion = "1.1.0"

// App exposes backend capabilities to the Wails React frontend.
type App struct {
	ctx        context.Context
	searchMu   sync.Mutex
	cancelFunc context.CancelFunc
	searching  bool
}

// NewApp creates an App instance.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetSystemInfo returns runtime OS/Arch and app metadata.
func (a *App) GetSystemInfo() model.SystemInfo {
	home, _ := os.UserHomeDir()
	return model.SystemInfo{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		AppVersion:   AppVersion,
		AuditLogPath: fileops.AuditLogPath(),
		HomeDir:      home,
	}
}

// SelectDirectory opens the native OS directory picker.
func (a *App) SelectDirectory(defaultDir string) (string, error) {
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:            "选择要搜索的目录",
		DefaultDirectory: defaultDir,
	})
}

// FilterDirectories validates dropped OS paths and returns deduplicated existing directories.
func (a *App) FilterDirectories(paths []string) []string {
	dirs := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, raw := range paths {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		cleaned := filepath.Clean(p)
		if _, dup := seen[cleaned]; dup {
			continue
		}
		info, err := os.Stat(cleaned)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[cleaned] = struct{}{}
		dirs = append(dirs, cleaned)
	}
	return dirs
}

// SelectQuarantineDir opens the native OS directory picker for quarantine destination.
func (a *App) SelectQuarantineDir(defaultDir string) (string, error) {
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:            "选择隔离备份目录",
		DefaultDirectory: defaultDir,
	})
}

// StartSearch launches an asynchronous concurrent search and streams results via Wails events.
func (a *App) StartSearch(opts model.SearchOptions) error {
	a.searchMu.Lock()
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
	searchCtx, cancel := context.WithCancel(a.ctx)
	a.cancelFunc = cancel
	a.searching = true
	a.searchMu.Unlock()

	go func() {
		defer func() {
			a.searchMu.Lock()
			a.searching = false
			a.searchMu.Unlock()
		}()

		summary := scanner.Run(searchCtx, opts, scanner.Callbacks{
			OnMatch: func(res model.SearchResult) {
				wailsRuntime.EventsEmit(a.ctx, "search:match", res)
			},
			OnProgress: func(prog model.SearchProgress) {
				wailsRuntime.EventsEmit(a.ctx, "search:progress", prog)
			},
		})
		wailsRuntime.EventsEmit(a.ctx, "search:done", summary)
	}()

	return nil
}

// CancelSearch cancels the currently running search.
func (a *App) CancelSearch() {
	a.searchMu.Lock()
	defer a.searchMu.Unlock()
	if a.cancelFunc != nil {
		a.cancelFunc()
		a.cancelFunc = nil
	}
}

// RevealInFolder opens the OS file manager and highlights the given file.
func (a *App) RevealInFolder(filePath string) error {
	return fileops.RevealInFolder(filePath)
}

// OpenFile opens the given file with the OS default application.
func (a *App) OpenFile(filePath string) error {
	return fileops.OpenFile(filePath)
}

// OpenURL opens an external URL in the user's default browser.
func (a *App) OpenURL(url string) {
	if url == "" {
		return
	}
	wailsRuntime.BrowserOpenURL(a.ctx, url)
}

// DeleteFiles deletes or quarantines selected files safely and writes audit logs.
func (a *App) DeleteFiles(req model.DeleteRequest) model.DeleteResponse {
	return fileops.DeleteFiles(req)
}

// GetAuditLogs returns the most recent audit log entries.
func (a *App) GetAuditLogs(limit int) ([]model.AuditEntry, error) {
	return fileops.GetAuditLogs(limit)
}

// ClearAuditLogs deletes all stored audit logs.
func (a *App) ClearAuditLogs() error {
	return fileops.ClearAuditLogs()
}

// ExportAuditCSV lets the user pick a destination file and saves audit logs as UTF-8 BOM CSV.
func (a *App) ExportAuditCSV() (string, error) {
	dest, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "导出操作审计日志",
		DefaultFilename: "DocPurger_审计日志.csv",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "CSV 表格 (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil || dest == "" {
		return "", err
	}
	if err := fileops.ExportAuditCSV(dest); err != nil {
		return "", fmt.Errorf("导出失败: %w", err)
	}
	return dest, nil
}
