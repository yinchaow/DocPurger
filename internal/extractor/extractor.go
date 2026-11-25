package extractor

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
)

const (
	// MaxArchiveEntryBytes caps a single in-memory extracted archive entry (32MB) to prevent zip bombs.
	MaxArchiveEntryBytes = 32 * 1024 * 1024
	// MaxArchiveTotalBytes caps total in-memory bytes extracted per archive (256MB) to prevent zip bombs.
	MaxArchiveTotalBytes = 256 * 1024 * 1024
	// MaxArchiveEntries caps total entries inspected per archive.
	MaxArchiveEntries = 2000
)

// Magic headers
var (
	magicZIP  = []byte{0x50, 0x4B, 0x03, 0x04}
	magicOLE2 = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	magicPDF  = []byte("%PDF-")
	magicRTF  = []byte("{\\rtf")
)

// ArchiveItem represents a single virtual file read from inside an archive.
type ArchiveItem struct {
	InnerPath string
	FileName  string
	Ext       string
	Size      int64
	Data      []byte
}

// ClassifyCategory maps a lowercase extension to a UI filter category.
func ClassifyCategory(ext string) string {
	switch strings.ToLower(ext) {
	case ".doc", ".docx", ".wps", ".rtf", ".odt":
		return "word"
	case ".xls", ".xlsx", ".et", ".ods":
		return "excel"
	case ".ppt", ".pptx", ".dps", ".odp":
		return "ppt"
	case ".pdf":
		return "pdf"
	case ".ofd":
		return "ofd"
	case ".zip", ".rar", ".7z", ".tar", ".gz", ".tgz":
		return "archive"
	case ".txt", ".md", ".markdown", ".csv", ".tsv", ".json", ".xml", ".yaml", ".yml",
		".ini", ".conf", ".cfg", ".log", ".html", ".htm", ".mht", ".mhtml",
		".sql", ".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".java", ".c", ".cpp", ".h", ".hpp",
		".cs", ".rs", ".sh", ".bat", ".ps1", ".css", ".scss", ".vue":
		return "text"
	default:
		return "other"
	}
}

// IsArchiveExt reports whether the extension is a supported archive container.
func IsArchiveExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".zip", ".rar", ".7z", ".tar", ".tgz", ".gz":
		return true
	default:
		return false
	}
}

// IsTextSearchableExt reports whether the extension is supported for content extraction.
func IsTextSearchableExt(ext string) bool {
	cat := ClassifyCategory(ext)
	return cat == "word" || cat == "excel" || cat == "ppt" || cat == "pdf" || cat == "ofd" || cat == "text"
}

// ExtractBytes extracts text from a document byte slice using magic-byte sniffing first,
// then extension routing. Returns (text, skipReason, err).
// skipReason is human-readable Chinese when content cannot be inspected (e.g. encrypted/scanned PDF).
func ExtractBytes(data []byte, fileName string) (text string, skipReason string, err error) {
	if len(data) == 0 {
		return "", "", nil
	}
	ext := strings.ToLower(filepath.Ext(fileName))

	// Trim leading whitespace for text-based magic checks (RTF/HTML/PDF)
	trimmed := bytes.TrimLeft(data, " \t\r\n\xef\xbb\xbf")

	// 1. Magic byte sniffing (solves fake .doc/.xls files that are actually RTF, HTML, MHT, DOCX, etc.)
	if bytes.HasPrefix(data, magicPDF) || bytes.HasPrefix(trimmed, magicPDF) {
		txt, pdfErr := ExtractPDF(data)
		if errors.Is(pdfErr, ErrEncryptedPDF) {
			return "", "加密 PDF，无法读取内容", nil
		}
		if errors.Is(pdfErr, ErrScannedPDF) {
			return "", "扫描件/纯图片 PDF（无文本层）", nil
		}
		if pdfErr != nil {
			return "", "PDF 解析失败", pdfErr
		}
		return txt, "", nil
	}

	if bytes.HasPrefix(trimmed, magicRTF) {
		return ExtractRTFText(trimmed), "", nil
	}

	if bytes.HasPrefix(data, magicZIP) {
		txt, kind, zipErr := ExtractOOXMLOrOFD(data, ext)
		if zipErr == nil && kind != "zip" {
			return txt, "", nil
		}
		// If extension is .docx/.xlsx/.pptx/.ofd but failed to open as zip, could be encrypted/corrupt
		if ext == ".docx" || ext == ".xlsx" || ext == ".pptx" || ext == ".ofd" {
			return "", "损坏或非标准压缩文档", zipErr
		}
	}

	if bytes.HasPrefix(data, magicOLE2) {
		// Encrypted OOXML (.docx/.xlsx/.pptx) is actually wrapped inside an OLE2 container with an EncryptedPackage stream!
		if ext == ".docx" || ext == ".xlsx" || ext == ".pptx" {
			return "", "加密 Office 文档，无法读取内容", nil
		}
		txt, oleErr := ExtractOLE2(data, ext)
		if errors.Is(oleErr, ErrEncryptedDocument) {
			return "", "加密 Office 文档，无法读取内容", nil
		}
		if oleErr != nil {
			return "", "旧版 Office 文档解析失败", oleErr
		}
		return txt, "", nil
	}

	// Check for HTML / MHT / XML saved as .doc or .xls
	if looksLikeHTMLOrMHT(trimmed) {
		return ExtractHTMLText(data), "", nil
	}

	// 2. Extension-based fallback for plain text / code / markdown / csv
	if ClassifyCategory(ext) == "text" || ext == ".wps" || ext == ".doc" || ext == ".xls" {
		if looksBinary(data) {
			return "", "", nil
		}
		return DecodeTextBytes(data), "", nil
	}

	return "", "", nil
}

func looksLikeHTMLOrMHT(trimmed []byte) bool {
	if len(trimmed) < 10 {
		return false
	}
	headLen := 512
	if len(trimmed) < headLen {
		headLen = len(trimmed)
	}
	head := strings.ToLower(string(trimmed[:headLen]))
	return strings.HasPrefix(head, "<!doctype html") ||
		strings.HasPrefix(head, "<html") ||
		strings.Contains(head, "<html") ||
		strings.Contains(head, "mime-version:") ||
		strings.HasPrefix(head, "<?xml") && (strings.Contains(head, "worddocument") || strings.Contains(head, "workbook") || strings.Contains(head, "mso-application"))
}

func looksBinary(data []byte) bool {
	sampleLen := 1024
	if len(data) < sampleLen {
		sampleLen = len(data)
	}
	nullCount := 0
	for i := 0; i < sampleLen; i++ {
		if data[i] == 0 {
			nullCount++
		}
	}
	// UTF-16 has many nulls; check BOM first before rejecting
	if len(data) >= 2 && ((data[0] == 0xFF && data[1] == 0xFE) || (data[0] == 0xFE && data[1] == 0xFF)) {
		return false
	}
	return nullCount > sampleLen/10
}

// WalkArchiveFile opens an archive from disk and invokes callback for each readable entry (recursively up to maxDepth).
// Returns skipReason if archive is password-protected or unreadable.
func WalkArchiveFile(diskPath string, maxDepth int, cb func(item ArchiveItem)) (skipReason string) {
	if maxDepth <= 0 {
		maxDepth = 3
	}
	ext := strings.ToLower(filepath.Ext(diskPath))
	lower := strings.ToLower(diskPath)

	switch {
	case ext == ".zip":
		return walkZipFile(diskPath, "", maxDepth, cb)
	case ext == ".7z":
		return walk7zFile(diskPath, "", maxDepth, cb)
	case ext == ".rar":
		return walkRarFile(diskPath, "", maxDepth, cb)
	case strings.HasSuffix(lower, ".tar.gz") || ext == ".tgz":
		f, err := os.Open(diskPath)
		if err != nil {
			return "无法打开压缩包"
		}
		defer f.Close()
		return walkTarGzReader(f, "", maxDepth, cb)
	case ext == ".tar":
		f, err := os.Open(diskPath)
		if err != nil {
			return "无法打开压缩包"
		}
		defer f.Close()
		return walkTarReader(f, "", maxDepth, cb)
	}
	return ""
}

func walkArchiveBytes(data []byte, archiveName string, prefix string, depthLeft int, cb func(item ArchiveItem)) string {
	if depthLeft <= 0 || len(data) == 0 {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(archiveName))
	lower := strings.ToLower(archiveName)

	switch {
	case ext == ".zip":
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return "嵌套 ZIP 损坏或无法读取"
		}
		return walkZipReader(zr, prefix, depthLeft, cb)
	case ext == ".7z":
		zr, err := sevenzip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			if isPasswordError(err) {
				return "加密 7z 压缩包"
			}
			return "嵌套 7z 无法读取"
		}
		return walk7zReader(zr, prefix, depthLeft, cb)
	case ext == ".rar":
		return walkRarReader(bytes.NewReader(data), prefix, depthLeft, cb)
	case strings.HasSuffix(lower, ".tar.gz") || ext == ".tgz":
		return walkTarGzReader(bytes.NewReader(data), prefix, depthLeft, cb)
	case ext == ".tar":
		return walkTarReader(bytes.NewReader(data), prefix, depthLeft, cb)
	}
	return ""
}

func walkZipFile(path string, prefix string, depthLeft int, cb func(item ArchiveItem)) string {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "ZIP 文件损坏或无法读取"
	}
	defer zr.Close()
	return walkZipReader(&zr.Reader, prefix, depthLeft, cb)
}

func walkZipReader(zr *zip.Reader, prefix string, depthLeft int, cb func(item ArchiveItem)) string {
	encryptedCount := 0
	count := 0
	var totalBytes int64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		count++
		if count > MaxArchiveEntries {
			break
		}

		// Fix Windows GBK filenames in ZIP entries (when UTF-8 flag 0x800 is not set)
		name := f.Name
		if (f.Flags & 0x800) == 0 {
			name = DecodeGBKString(name)
		}
		fullInner := joinInnerPath(prefix, name)
		baseName := filepath.Base(name)
		ext := strings.ToLower(filepath.Ext(baseName))

		// Bit 0 of Flags indicates encrypted entry
		if (f.Flags & 0x1) != 0 {
			encryptedCount++
			// Still emit filename-only item so filename search can match!
			cb(ArchiveItem{
				InnerPath: fullInner,
				FileName:  baseName,
				Ext:       ext,
				Size:      int64(f.UncompressedSize64),
			})
			continue
		}

		var entryData []byte
		if totalBytes < MaxArchiveTotalBytes && f.UncompressedSize64 <= MaxArchiveEntryBytes && (IsTextSearchableExt(ext) || (depthLeft > 1 && IsArchiveExt(ext))) {
			rc, err := f.Open()
			if err == nil {
				rem := MaxArchiveTotalBytes - totalBytes
				limit := int64(MaxArchiveEntryBytes)
				if rem < limit {
					limit = rem
				}
				entryData, _ = readLimited(rc, limit)
				totalBytes += int64(len(entryData))
				_ = rc.Close()
			}
		}

		if depthLeft > 1 && IsArchiveExt(ext) && len(entryData) > 0 {
			walkArchiveBytes(entryData, baseName, fullInner, depthLeft-1, cb)
			continue
		}

		cb(ArchiveItem{
			InnerPath: fullInner,
			FileName:  baseName,
			Ext:       ext,
			Size:      int64(f.UncompressedSize64),
			Data:      entryData,
		})
	}
	if encryptedCount > 0 {
		return "包含加密条目的 ZIP 压缩包"
	}
	return ""
}

func walk7zFile(path string, prefix string, depthLeft int, cb func(item ArchiveItem)) string {
	zr, err := sevenzip.OpenReader(path)
	if err != nil {
		if isPasswordError(err) {
			return "加密 7z 压缩包"
		}
		return "7z 压缩包无法读取"
	}
	defer zr.Close()
	return walk7zReader(&zr.Reader, prefix, depthLeft, cb)
}

func walk7zReader(zr *sevenzip.Reader, prefix string, depthLeft int, cb func(item ArchiveItem)) string {
	encrypted := false
	count := 0
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		count++
		if count > MaxArchiveEntries {
			break
		}

		name := DecodeGBKString(f.Name)
		fullInner := joinInnerPath(prefix, name)
		baseName := filepath.Base(name)
		ext := strings.ToLower(filepath.Ext(baseName))
		size := int64(f.UncompressedSize)

		var entryData []byte
		if size <= MaxArchiveEntryBytes && (IsTextSearchableExt(ext) || (depthLeft > 1 && IsArchiveExt(ext))) {
			rc, err := f.Open()
			if err != nil {
				if isPasswordError(err) {
					encrypted = true
				}
			} else {
				entryData, err = readLimited(rc, MaxArchiveEntryBytes)
				if err != nil && isPasswordError(err) {
					encrypted = true
					entryData = nil
				}
				_ = rc.Close()
			}
		}

		if depthLeft > 1 && IsArchiveExt(ext) && len(entryData) > 0 {
			walkArchiveBytes(entryData, baseName, fullInner, depthLeft-1, cb)
			continue
		}

		cb(ArchiveItem{
			InnerPath: fullInner,
			FileName:  baseName,
			Ext:       ext,
			Size:      size,
			Data:      entryData,
		})
	}
	if encrypted {
		return "加密 7z 压缩包"
	}
	return ""
}

func walkRarFile(path string, prefix string, depthLeft int, cb func(item ArchiveItem)) string {
	f, err := os.Open(path)
	if err != nil {
		return "RAR 压缩包无法打开"
	}
	defer f.Close()
	return walkRarReader(f, prefix, depthLeft, cb)
}

func walkRarReader(r io.Reader, prefix string, depthLeft int, cb func(item ArchiveItem)) string {
	rr, err := rardecode.NewReader(r)
	if err != nil {
		if isPasswordError(err) {
			return "加密 RAR 压缩包"
		}
		return "RAR 压缩包无法读取"
	}

	encrypted := false
	count := 0
	for {
		hdr, err := rr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			if isPasswordError(err) {
				encrypted = true
			}
			break
		}
		if hdr.IsDir {
			continue
		}
		count++
		if count > MaxArchiveEntries {
			break
		}

		name := DecodeGBKString(hdr.Name)
		fullInner := joinInnerPath(prefix, name)
		baseName := filepath.Base(name)
		ext := strings.ToLower(filepath.Ext(baseName))

		var entryData []byte
		if hdr.UnPackedSize <= MaxArchiveEntryBytes && (IsTextSearchableExt(ext) || (depthLeft > 1 && IsArchiveExt(ext))) {
			entryData, err = readLimited(rr, MaxArchiveEntryBytes)
			if err != nil && isPasswordError(err) {
				encrypted = true
				entryData = nil
			}
		}

		if depthLeft > 1 && IsArchiveExt(ext) && len(entryData) > 0 {
			walkArchiveBytes(entryData, baseName, fullInner, depthLeft-1, cb)
			continue
		}

		cb(ArchiveItem{
			InnerPath: fullInner,
			FileName:  baseName,
			Ext:       ext,
			Size:      hdr.UnPackedSize,
			Data:      entryData,
		})
	}
	if encrypted {
		return "加密 RAR 压缩包"
	}
	return ""
}

func walkTarGzReader(r io.Reader, prefix string, depthLeft int, cb func(item ArchiveItem)) string {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return "GZIP 压缩包损坏或无法读取"
	}
	defer gr.Close()
	return walkTarReader(gr, prefix, depthLeft, cb)
}

func walkTarReader(r io.Reader, prefix string, depthLeft int, cb func(item ArchiveItem)) string {
	tr := tar.NewReader(r)
	count := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		count++
		if count > MaxArchiveEntries {
			break
		}

		name := DecodeGBKString(hdr.Name)
		fullInner := joinInnerPath(prefix, name)
		baseName := filepath.Base(name)
		ext := strings.ToLower(filepath.Ext(baseName))

		var entryData []byte
		if hdr.Size <= MaxArchiveEntryBytes && (IsTextSearchableExt(ext) || (depthLeft > 1 && IsArchiveExt(ext))) {
			entryData, _ = readLimited(tr, MaxArchiveEntryBytes)
		}

		if depthLeft > 1 && IsArchiveExt(ext) && len(entryData) > 0 {
			walkArchiveBytes(entryData, baseName, fullInner, depthLeft-1, cb)
			continue
		}

		cb(ArchiveItem{
			InnerPath: fullInner,
			FileName:  baseName,
			Ext:       ext,
			Size:      hdr.Size,
			Data:      entryData,
		})
	}
	return ""
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	lr := io.LimitReader(r, limit+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return data[:limit], nil
	}
	return data, nil
}

func joinInnerPath(prefix, name string) string {
	clean := strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "/")
	if prefix == "" {
		return clean
	}
	return prefix + " → " + clean
}

func isPasswordError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "password") || strings.Contains(s, "encrypt") || strings.Contains(s, "bad password")
}
