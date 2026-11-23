package model

import "time"

// MatchMode defines boolean combination for multiple keywords.
type MatchMode string

const (
	MatchModeAnd MatchMode = "AND"
	MatchModeOr  MatchMode = "OR"
)

// SearchScope controls whether filename, content, or both are searched.
type SearchScope string

const (
	SearchScopeBoth     SearchScope = "both"
	SearchScopeFilename SearchScope = "filename"
	SearchScopeContent  SearchScope = "content"
)

// DeleteMode controls how selected files are removed.
type DeleteMode string

const (
	DeleteModeTrash      DeleteMode = "trash"
	DeleteModeQuarantine DeleteMode = "quarantine"
	DeleteModePermanent  DeleteMode = "permanent"
)

// SearchOptions holds user search configuration.
type SearchOptions struct {
	RootDir         string      `json:"rootDir"`         // Backward-compatible single root dir
	RootDirs        []string    `json:"rootDirs"`        // Multi-directory search targets
	Keywords        []string    `json:"keywords"`
	ExcludeKeywords []string    `json:"excludeKeywords"`
	Mode            MatchMode   `json:"mode"`            // "AND" or "OR"
	Scope           SearchScope `json:"scope"`           // "both", "filename", "content"
	CaseSensitive   bool        `json:"caseSensitive"`
	IncludeArchives bool        `json:"includeArchives"`
	Categories      []string    `json:"categories"`      // e.g. ["word","excel","ppt","pdf","ofd","text","archive"]
	MaxFileSizeMB   int64       `json:"maxFileSizeMB"`   // 0 means default (100MB)
	IncludeHidden   bool        `json:"includeHidden"`
	MaxArchiveDepth int         `json:"maxArchiveDepth"` // default 2
}

// HighlightSpan marks a rune-based [Start, End) span inside snippet text.
type HighlightSpan struct {
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Keyword string `json:"keyword"`
}

// MatchSnippet is a contextual excerpt around a keyword match.
type MatchSnippet struct {
	Source     string          `json:"source"` // "filename" or "content"
	Text       string          `json:"text"`
	Highlights []HighlightSpan `json:"highlights"`
}

// SearchResult represents a single matched file or archive entry.
type SearchResult struct {
	ID              string         `json:"id"`
	FilePath        string         `json:"filePath"`        // Real file path on disk (for archives, the outer archive path)
	RelativePath    string         `json:"relativePath"`    // Path relative to its root dir
	RootLabel       string         `json:"rootLabel"`       // Base folder name when searching multiple directories
	FileName        string         `json:"fileName"`        // Display filename (inner filename if inside archive)
	ArchivePath     string         `json:"archivePath"`     // Non-empty if inside archive (e.g. outer archive path)
	InnerPath       string         `json:"innerPath"`       // Path inside archive, e.g. "财务/报告.docx"
	IsArchiveEntry  bool           `json:"isArchiveEntry"`
	Ext             string         `json:"ext"`             // Lowercase extension with dot, e.g. ".docx"
	Category        string         `json:"category"`        // word, excel, ppt, pdf, ofd, text, archive, other
	Size            int64          `json:"size"`            // Disk file size (or entry size)
	DiskFileSize    int64          `json:"diskFileSize"`    // Actual disk file size (important for archive entries)
	ModTime         string         `json:"modTime"`         // Formatted timestamp
	ModTimeUnix     int64          `json:"modTimeUnix"`
	MatchInFilename bool           `json:"matchInFilename"`
	MatchInContent  bool           `json:"matchInContent"`
	MatchedKeywords []string       `json:"matchedKeywords"`
	MatchCount      int            `json:"matchCount"`
	Snippets        []MatchSnippet `json:"snippets"`
}

// SkippedFile records files whose content could not be inspected (e.g., encrypted/scanned PDF, encrypted archive).
type SkippedFile struct {
	FilePath     string `json:"filePath"`
	RelativePath string `json:"relativePath"`
	Reason       string `json:"reason"`
}

// SearchProgress is streamed to frontend during scanning.
type SearchProgress struct {
	ScannedFiles int    `json:"scannedFiles"`
	MatchedCount int    `json:"matchedCount"`
	SkippedCount int    `json:"skippedCount"`
	CurrentDir   string `json:"currentDir"`
	ElapsedMs    int64  `json:"elapsedMs"`
	Running      bool   `json:"running"`
}

// SearchSummary is emitted when a search finishes or is cancelled.
type SearchSummary struct {
	ScannedFiles int           `json:"scannedFiles"`
	MatchedCount int           `json:"matchedCount"`
	SkippedCount int           `json:"skippedCount"`
	SkippedFiles []SkippedFile `json:"skippedFiles"`
	ElapsedMs    int64         `json:"elapsedMs"`
	Cancelled    bool          `json:"cancelled"`
	Error        string        `json:"error,omitempty"`
}

// DeleteRequest describes a batch delete/quarantine operation.
type DeleteRequest struct {
	Paths         []string   `json:"paths"`         // Physical disk paths (deduplicated)
	Mode          DeleteMode `json:"mode"`          // "trash", "quarantine", "permanent"
	QuarantineDir string     `json:"quarantineDir"` // Required when mode == "quarantine"
	RootDir       string     `json:"rootDir"`       // Legacy single root dir
	RootDirs      []string   `json:"rootDirs"`      // Multi-root dirs for quarantine relative structure
	Keywords      []string   `json:"keywords"`      // Recorded in audit log
}

// DeleteItemResult is the per-file outcome.
type DeleteItemResult struct {
	Path        string `json:"path"`
	Success     bool   `json:"success"`
	Destination string `json:"destination,omitempty"`
	Error       string `json:"error,omitempty"`
}

// DeleteResponse summarizes a batch delete operation.
type DeleteResponse struct {
	Mode         DeleteMode         `json:"mode"`
	Total        int                `json:"total"`
	Succeeded    int                `json:"succeeded"`
	Failed       int                `json:"failed"`
	FreedBytes   int64              `json:"freedBytes"`
	Results      []DeleteItemResult `json:"results"`
	AuditLogPath string             `json:"auditLogPath"`
}

// AuditEntry is persisted in the local audit log for traceability.
type AuditEntry struct {
	Timestamp   time.Time  `json:"timestamp"`
	Mode        DeleteMode `json:"mode"`
	FilePath    string     `json:"filePath"`
	Destination string     `json:"destination,omitempty"`
	FileSize    int64      `json:"fileSize"`
	Keywords    []string   `json:"keywords"`
	Success     bool       `json:"success"`
	Error       string     `json:"error,omitempty"`
}

// SystemInfo exposes platform capabilities to the frontend.
type SystemInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	AppVersion   string `json:"appVersion"`
	AuditLogPath string `json:"auditLogPath"`
	HomeDir      string `json:"homeDir"`
}
