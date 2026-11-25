package fileops

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"docpurger/internal/model"
)

var auditMu sync.Mutex

// AuditLogPath returns the persistent JSONL audit log file path in user config dir.
func AuditLogPath() string {
	cfgDir, err := os.UserConfigDir()
	if err != nil || cfgDir == "" {
		home, _ := os.UserHomeDir()
		cfgDir = filepath.Join(home, ".config")
	}
	dir := filepath.Join(cfgDir, "DocPurger")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "audit.jsonl")
}

// DeleteFiles performs safe deduplicated deletion (trash / quarantine / permanent) and records audit entries.
func DeleteFiles(req model.DeleteRequest) model.DeleteResponse {
	mode := req.Mode
	if mode != model.DeleteModeQuarantine && mode != model.DeleteModePermanent {
		mode = model.DeleteModeTrash
	}

	// Deduplicate real disk paths (critical when multiple entries inside the same archive are selected)
	seen := make(map[string]bool, len(req.Paths))
	uniquePaths := make([]string, 0, len(req.Paths))
	for _, p := range req.Paths {
		clean := filepath.Clean(strings.TrimSpace(p))
		if clean == "" || clean == "." {
			continue
		}
		if !seen[clean] {
			seen[clean] = true
			uniquePaths = append(uniquePaths, clean)
		}
	}

	resp := model.DeleteResponse{
		Mode:         mode,
		Total:        len(uniquePaths),
		Results:      make([]model.DeleteItemResult, 0, len(uniquePaths)),
		AuditLogPath: AuditLogPath(),
	}

	if mode == model.DeleteModeQuarantine {
		qDir := strings.TrimSpace(req.QuarantineDir)
		if qDir == "" {
			for _, p := range uniquePaths {
				resp.Results = append(resp.Results, model.DeleteItemResult{
					Path:  p,
					Error: "未指定隔离目录",
				})
				resp.Failed++
			}
			return resp
		}
		if isQuarantineInsideRoots(qDir, req.RootDir, req.RootDirs) {
			for _, p := range uniquePaths {
				resp.Results = append(resp.Results, model.DeleteItemResult{
					Path:  p,
					Error: "安全拦截：隔离目录不能位于待搜索目录内部",
				})
				resp.Failed++
			}
			return resp
		}
		if err := os.MkdirAll(qDir, 0o755); err != nil {
			for _, p := range uniquePaths {
				resp.Results = append(resp.Results, model.DeleteItemResult{
					Path:  p,
					Error: "无法创建隔离目录: " + err.Error(),
				})
				resp.Failed++
			}
			return resp
		}
	}

	var auditBatch []model.AuditEntry
	for _, p := range uniquePaths {
		if isProtectedSystemPath(p) {
			itemRes := model.DeleteItemResult{
				Path:  p,
				Error: "安全拦截：禁止删除系统关键路径或用户主目录",
			}
			resp.Results = append(resp.Results, itemRes)
			resp.Failed++
			auditBatch = append(auditBatch, model.AuditEntry{
				Timestamp: time.Now(),
				Mode:      mode,
				FilePath:  p,
				Keywords:  req.Keywords,
				Success:   false,
				Error:     itemRes.Error,
			})
			continue
		}
		var fileSize int64
		if fi, err := os.Stat(p); err == nil {
			fileSize = fi.Size()
		} else {
			itemRes := model.DeleteItemResult{
				Path:  p,
				Error: "文件不存在或已被移除",
			}
			resp.Results = append(resp.Results, itemRes)
			resp.Failed++
			auditBatch = append(auditBatch, model.AuditEntry{
				Timestamp: time.Now(),
				Mode:      mode,
				FilePath:  p,
				Keywords:  req.Keywords,
				Success:   false,
				Error:     itemRes.Error,
			})
			continue
		}

		var (
			dest string
			err  error
		)
		switch mode {
		case model.DeleteModeTrash:
			err = MoveToTrash(p)
		case model.DeleteModeQuarantine:
			dest, err = moveToQuarantine(p, req.RootDir, req.RootDirs, req.QuarantineDir)
		case model.DeleteModePermanent:
			err = os.Remove(p)
		}

		itemRes := model.DeleteItemResult{
			Path:        p,
			Success:     err == nil,
			Destination: dest,
		}
		if err != nil {
			itemRes.Error = err.Error()
			resp.Failed++
		} else {
			resp.Succeeded++
			resp.FreedBytes += fileSize
		}
		resp.Results = append(resp.Results, itemRes)

		auditBatch = append(auditBatch, model.AuditEntry{
			Timestamp:   time.Now(),
			Mode:        mode,
			FilePath:    p,
			Destination: dest,
			FileSize:    fileSize,
			Keywords:    req.Keywords,
			Success:     err == nil,
			Error:       itemRes.Error,
		})
	}

	_ = appendAuditEntries(auditBatch)
	return resp
}

func moveToQuarantine(srcPath, rootDir string, rootDirs []string, quarantineDir string) (string, error) {
	roots := make([]string, 0, len(rootDirs)+1)
	roots = append(roots, rootDirs...)
	if strings.TrimSpace(rootDir) != "" {
		roots = append(roots, rootDir)
	}
	multiRoot := len(roots) > 1

	var targetRel string
	for _, r := range roots {
		cleanRoot := filepath.Clean(strings.TrimSpace(r))
		if cleanRoot == "" {
			continue
		}
		if rel, err := filepath.Rel(cleanRoot, srcPath); err == nil && !strings.HasPrefix(rel, "..") {
			if multiRoot {
				base := filepath.Base(cleanRoot)
				if base != "" && base != "." && base != string(filepath.Separator) {
					targetRel = filepath.Join(base, rel)
					break
				}
			}
			targetRel = rel
			break
		}
	}
	if targetRel == "" {
		targetRel = filepath.Base(srcPath)
	}

	destPath := filepath.Join(quarantineDir, targetRel)
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", err
	}
	destPath = ensureUniquePath(destPath)

	if err := moveFileCrossDevice(srcPath, destPath); err != nil {
		return "", err
	}
	return destPath, nil
}

func ensureUniquePath(dest string) string {
	if _, err := os.Stat(dest); errors.Is(err, os.ErrNotExist) {
		return dest
	}
	dir := filepath.Dir(dest)
	ext := filepath.Ext(dest)
	base := strings.TrimSuffix(filepath.Base(dest), ext)
	stamp := time.Now().Format("20060102_150405")
	for i := 1; i <= 9999; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s_%s_%d%s", base, stamp, i, ext))
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
	return filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, time.Now().UnixNano(), ext))
}

func moveFileCrossDevice(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Cross-device move fallback: copy then remove
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	fi, err := in.Stat()
	if err != nil {
		_ = in.Close()
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode())
	if err != nil {
		_ = in.Close()
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = in.Close()
		_ = out.Close()
		_ = os.Remove(dst)
		return err
	}
	_ = in.Close()
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return os.Remove(src)
}

func appendAuditEntries(entries []model.AuditEntry) error {
	if len(entries) == 0 {
		return nil
	}
	auditMu.Lock()
	defer auditMu.Unlock()

	f, err := os.OpenFile(AuditLogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, e := range entries {
		_ = enc.Encode(e)
	}
	return nil
}

// GetAuditLogs reads the latest N audit entries (newest first).
func GetAuditLogs(limit int) ([]model.AuditEntry, error) {
	if limit <= 0 {
		limit = 200
	}
	auditMu.Lock()
	defer auditMu.Unlock()

	f, err := os.Open(AuditLogPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []model.AuditEntry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var all []model.AuditEntry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var entry model.AuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			all = append(all, entry)
		}
	}

	// Reverse so newest comes first
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

// ClearAuditLogs truncates the audit log file.
func ClearAuditLogs() error {
	auditMu.Lock()
	defer auditMu.Unlock()
	return os.Remove(AuditLogPath())
}

// ExportAuditCSV exports audit logs to a UTF-8 BOM CSV file (Excel-friendly on Windows & Kylin).
func ExportAuditCSV(destPath string) error {
	entries, err := GetAuditLogs(10000)
	if err != nil {
		return err
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// UTF-8 BOM so Windows Excel / WPS opens Chinese cleanly
	_, _ = f.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(f)
	defer w.Flush()

	_ = w.Write([]string{"时间", "操作方式", "文件路径", "目标位置", "文件大小(字节)", "命中关键词", "结果", "错误信息"})
	for _, e := range entries {
		status := "成功"
		if !e.Success {
			status = "失败"
		}
		modeLabel := string(e.Mode)
		switch e.Mode {
		case model.DeleteModeTrash:
			modeLabel = "移入回收站"
		case model.DeleteModeQuarantine:
			modeLabel = "移至隔离目录"
		case model.DeleteModePermanent:
			modeLabel = "彻底删除"
		}
		_ = w.Write([]string{
			e.Timestamp.Format("2006-01-02 15:04:05"),
			modeLabel,
			e.FilePath,
			e.Destination,
			strconv.FormatInt(e.FileSize, 10),
			strings.Join(e.Keywords, "; "),
			status,
			e.Error,
		})
	}
	return nil
}

func isQuarantineInsideRoots(quarantineDir, rootDir string, rootDirs []string) bool {
	absQ, err := filepath.Abs(filepath.Clean(quarantineDir))
	if err != nil {
		return false
	}
	roots := make([]string, 0, len(rootDirs)+1)
	roots = append(roots, rootDirs...)
	if strings.TrimSpace(rootDir) != "" {
		roots = append(roots, rootDir)
	}
	for _, r := range roots {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		absR, err := filepath.Abs(filepath.Clean(r))
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(absR, absQ)
		if err == nil && (rel == "." || (!strings.HasPrefix(rel, "..") && rel != "")) {
			return true
		}
	}
	return false
}

func isProtectedSystemPath(targetPath string) bool {
	abs, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		return false
	}
	// Never allow deleting filesystem root or drive root (e.g. "/" or "C:\")
	if filepath.Dir(abs) == abs {
		return true
	}
	// Never allow deleting user home directory itself
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if absHome, err := filepath.Abs(filepath.Clean(home)); err == nil {
			if strings.EqualFold(abs, absHome) {
				return true
			}
		}
	}
	lower := strings.ToLower(filepath.ToSlash(abs))
	protectedExact := []string{
		"/bin", "/sbin", "/usr", "/etc", "/var", "/lib", "/lib64", "/boot", "/dev", "/proc", "/sys", "/root", "/home",
		"c:/windows", "c:/program files", "c:/program files (x86)", "c:/programdata", "c:/users",
	}
	for _, p := range protectedExact {
		if lower == p {
			return true
		}
	}
	return false
}
