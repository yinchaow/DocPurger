package scanner

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"docpurger/internal/extractor"
	"docpurger/internal/matcher"
	"docpurger/internal/model"
)

// Callbacks receive streaming search updates.
type Callbacks struct {
	OnMatch    func(res model.SearchResult)
	OnProgress func(prog model.SearchProgress)
}

var ignoredDirs = map[string]bool{
	".git":                      true,
	".svn":                      true,
	".hg":                       true,
	"node_modules":              true,
	"$recycle.bin":              true,
	"system volume information": true,
	"__pycache__":               true,
	".idea":                     true,
	".vscode":                   true,
	".trash":                    true,
	".trash-1000":               true,
}

type fileJob struct {
	diskPath     string
	relativePath string
	rootLabel    string
	fileName     string
	ext          string
	category     string
	size         int64
	modTime      time.Time
}

// normalizeRootDirs validates, deduplicates, and prunes child directories already covered by a parent root.
func normalizeRootDirs(opts model.SearchOptions) ([]string, string) {
	raw := make([]string, 0, len(opts.RootDirs)+1)
	raw = append(raw, opts.RootDirs...)
	if strings.TrimSpace(opts.RootDir) != "" {
		raw = append(raw, opts.RootDir)
	}

	seen := make(map[string]bool)
	valid := make([]string, 0, len(raw))
	for _, r := range raw {
		trimmed := strings.TrimSpace(r)
		if trimmed == "" {
			continue
		}
		abs, err := filepath.Abs(filepath.Clean(trimmed))
		if err != nil {
			continue
		}
		key := abs
		if runtime.GOOS == "windows" {
			key = strings.ToLower(abs)
		}
		if seen[key] {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[key] = true
		valid = append(valid, abs)
	}

	if len(valid) == 0 {
		if len(raw) == 0 {
			return nil, "请至少添加一个要搜索的目录"
		}
		return nil, "指定的搜索目录不存在或无法访问"
	}

	// Sort shortest paths first so parent directories precede their subdirectories
	sort.Slice(valid, func(i, j int) bool {
		return len(valid[i]) < len(valid[j])
	})

	pruned := make([]string, 0, len(valid))
	for _, candidate := range valid {
		covered := false
		for _, parent := range pruned {
			if isSubPath(parent, candidate) {
				covered = true
				break
			}
		}
		if !covered {
			pruned = append(pruned, candidate)
		}
	}
	return pruned, ""
}

func isSubPath(parent, target string) bool {
	p := filepath.Clean(parent)
	t := filepath.Clean(target)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
		t = strings.ToLower(t)
	}
	if p == t {
		return true
	}
	sep := string(filepath.Separator)
	if !strings.HasSuffix(p, sep) {
		p += sep
	}
	return strings.HasPrefix(t, p)
}

// Run executes a concurrent single- or multi-directory search and returns a summary when done or cancelled.
func Run(ctx context.Context, opts model.SearchOptions, cb Callbacks) model.SearchSummary {
	start := time.Now()
	summary := model.SearchSummary{}

	roots, errMsg := normalizeRootDirs(opts)
	if errMsg != "" {
		summary.Error = errMsg
		return summary
	}

	engine := matcher.New(opts)
	if !engine.HasKeywords() {
		summary.Error = "请至少输入一个搜索关键词"
		return summary
	}

	maxSizeBytes := opts.MaxFileSizeMB * 1024 * 1024
	if maxSizeBytes <= 0 {
		maxSizeBytes = 100 * 1024 * 1024 // Default 100MB
	}
	maxDepth := opts.MaxArchiveDepth
	if maxDepth <= 0 {
		maxDepth = 2
	}

	catFilter := make(map[string]bool, len(opts.Categories))
	for _, c := range opts.Categories {
		if strings.TrimSpace(c) != "" {
			catFilter[strings.ToLower(strings.TrimSpace(c))] = true
		}
	}

	multiRoot := len(roots) > 1

	var (
		scannedFiles int64
		matchedCount int64
		skippedMu    sync.Mutex
		skippedFiles []model.SkippedFile
		currentDir   atomic.Value
	)
	currentDir.Store(roots[0])

	addSkip := func(path, rel, reason string) {
		if reason == "" {
			return
		}
		skippedMu.Lock()
		if len(skippedFiles) < 500 {
			skippedFiles = append(skippedFiles, model.SkippedFile{
				FilePath:     path,
				RelativePath: rel,
				Reason:       reason,
			})
		}
		skippedMu.Unlock()
	}

	progressDone := make(chan struct{})
	if cb.OnProgress != nil {
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-progressDone:
					return
				case <-ticker.C:
					skippedMu.Lock()
					sc := len(skippedFiles)
					skippedMu.Unlock()
					dir, _ := currentDir.Load().(string)
					cb.OnProgress(model.SearchProgress{
						ScannedFiles: int(atomic.LoadInt64(&scannedFiles)),
						MatchedCount: int(atomic.LoadInt64(&matchedCount)),
						SkippedCount: sc,
						CurrentDir:   dir,
						ElapsedMs:    time.Since(start).Milliseconds(),
						Running:      true,
					})
				}
			}
		}()
	}

	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	if workers > 8 {
		workers = 8
	}

	jobs := make(chan fileJob, workers*16)
	var wg sync.WaitGroup

	emitMatch := func(res model.SearchResult) {
		atomic.AddInt64(&matchedCount, 1)
		if cb.OnMatch != nil {
			cb.OnMatch(res)
		}
	}

	processJob := func(job fileJob) {
		defer atomic.AddInt64(&scannedFiles, 1)

		isArchive := extractor.IsArchiveExt(job.ext)
		if isArchive {
			if !engine.QuickFilenameReject(job.fileName) {
				if out := engine.Evaluate(job.fileName, ""); out.Matched {
					emitMatch(buildResult(job, "", "", false, job.size, out))
				}
			}
			if opts.IncludeArchives && job.size <= maxSizeBytes {
				skipReason := extractor.WalkArchiveFile(job.diskPath, maxDepth, func(item extractor.ArchiveItem) {
					select {
					case <-ctx.Done():
						return
					default:
					}

					itemCat := extractor.ClassifyCategory(item.Ext)
					if len(catFilter) > 0 && !catFilter[itemCat] && !(len(catFilter) == 1 && catFilter["archive"]) {
						return
					}
					if engine.QuickFilenameReject(item.FileName) {
						return
					}

					var content string
					if engine.Scope() != model.SearchScopeFilename && len(item.Data) > 0 {
						txt, innerSkip, _ := extractor.ExtractBytes(item.Data, item.FileName)
						content = txt
						if innerSkip != "" {
							addSkip(job.diskPath, job.relativePath+" → "+item.InnerPath, innerSkip)
						}
					}

					out := engine.Evaluate(item.FileName, content)
					if out.Matched {
						innerJob := job
						innerJob.fileName = item.FileName
						innerJob.ext = item.Ext
						innerJob.category = itemCat
						emitMatch(buildResult(innerJob, job.diskPath, item.InnerPath, true, item.Size, out))
					}
				})
				if skipReason != "" {
					addSkip(job.diskPath, job.relativePath, skipReason)
				}
			}
			return
		}

		if len(catFilter) > 0 && !catFilter[job.category] {
			return
		}
		if engine.QuickFilenameReject(job.fileName) {
			return
		}

		if engine.Scope() == model.SearchScopeFilename {
			out := engine.Evaluate(job.fileName, "")
			if out.Matched {
				emitMatch(buildResult(job, "", "", false, job.size, out))
			}
			return
		}

		var content string
		if extractor.IsTextSearchableExt(job.ext) {
			if job.size > maxSizeBytes {
				addSkip(job.diskPath, job.relativePath, "文件体积超过扫描上限，仅检查文件名")
			} else if job.size > 0 {
				data, readErr := os.ReadFile(job.diskPath)
				if readErr != nil {
					addSkip(job.diskPath, job.relativePath, "无读取权限或文件被占用")
				} else {
					txt, skipReason, _ := extractor.ExtractBytes(data, job.fileName)
					content = txt
					if skipReason != "" {
						addSkip(job.diskPath, job.relativePath, skipReason)
					}
				}
			}
		}

		out := engine.Evaluate(job.fileName, content)
		if out.Matched {
			emitMatch(buildResult(job, "", "", false, job.size, out))
		}
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					processJob(job)
				}
			}
		}()
	}

	// Walk every deduplicated root directory
	for _, rootDir := range roots {
		if ctx.Err() != nil {
			break
		}
		rootLabel := ""
		if multiRoot {
			rootLabel = filepath.Base(rootDir)
			if rootLabel == "" || rootLabel == "." || rootLabel == string(filepath.Separator) {
				rootLabel = rootDir
			}
		}

		_ = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, walkErr error) error {
			select {
			case <-ctx.Done():
				return filepath.SkipAll
			default:
			}

			if walkErr != nil {
				return nil
			}

			name := d.Name()
			if d.IsDir() {
				if path != rootDir {
					lowerName := strings.ToLower(name)
					if ignoredDirs[lowerName] {
						return filepath.SkipDir
					}
					if !opts.IncludeHidden && strings.HasPrefix(name, ".") {
						return filepath.SkipDir
					}
				}
				currentDir.Store(path)
				return nil
			}

			if !opts.IncludeHidden && strings.HasPrefix(name, ".") {
				return nil
			}
			if strings.HasPrefix(name, "~$") {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(name))
			cat := extractor.ClassifyCategory(ext)

			if len(catFilter) > 0 && !catFilter[cat] {
				return nil
			}

			fi, statErr := d.Info()
			if statErr != nil || !fi.Mode().IsRegular() {
				return nil
			}

			rel, relErr := filepath.Rel(rootDir, path)
			if relErr != nil {
				rel = path
			}

			job := fileJob{
				diskPath:     path,
				relativePath: filepath.ToSlash(rel),
				rootLabel:    rootLabel,
				fileName:     name,
				ext:          ext,
				category:     cat,
				size:         fi.Size(),
				modTime:      fi.ModTime(),
			}

			select {
			case <-ctx.Done():
				return filepath.SkipAll
			case jobs <- job:
				return nil
			}
		})
	}

	close(jobs)
	wg.Wait()
	close(progressDone)

	skippedMu.Lock()
	finalSkipped := append([]model.SkippedFile(nil), skippedFiles...)
	skippedMu.Unlock()

	summary.ScannedFiles = int(atomic.LoadInt64(&scannedFiles))
	summary.MatchedCount = int(atomic.LoadInt64(&matchedCount))
	summary.SkippedCount = len(finalSkipped)
	summary.SkippedFiles = finalSkipped
	summary.ElapsedMs = time.Since(start).Milliseconds()
	summary.Cancelled = ctx.Err() != nil

	if cb.OnProgress != nil {
		cb.OnProgress(model.SearchProgress{
			ScannedFiles: summary.ScannedFiles,
			MatchedCount: summary.MatchedCount,
			SkippedCount: summary.SkippedCount,
			CurrentDir:   "",
			ElapsedMs:    summary.ElapsedMs,
			Running:      false,
		})
	}

	return summary
}

func buildResult(job fileJob, archivePath, innerPath string, isArchiveEntry bool, displaySize int64, out matcher.Outcome) model.SearchResult {
	sum := sha1.Sum([]byte(job.diskPath + "|" + innerPath))
	id := hex.EncodeToString(sum[:8])

	return model.SearchResult{
		ID:              id,
		FilePath:        job.diskPath,
		RelativePath:    job.relativePath,
		RootLabel:       job.rootLabel,
		FileName:        job.fileName,
		ArchivePath:     archivePath,
		InnerPath:       innerPath,
		IsArchiveEntry:  isArchiveEntry,
		Ext:             job.ext,
		Category:        job.category,
		Size:            displaySize,
		DiskFileSize:    job.size,
		ModTime:         job.modTime.Format("2006-01-02 15:04"),
		ModTimeUnix:     job.modTime.Unix(),
		MatchInFilename: out.MatchInFilename,
		MatchInContent:  out.MatchInContent,
		MatchedKeywords: out.MatchedKeywords,
		MatchCount:      out.TotalHits,
		Snippets:        out.Snippets,
	}
}
