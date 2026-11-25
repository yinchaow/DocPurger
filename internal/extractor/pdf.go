package extractor

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/ledongthuc/pdf"
)

var (
	ErrEncryptedPDF = errors.New("encrypted PDF")
	ErrScannedPDF   = errors.New("scanned or image-only PDF without text layer")
)

// ExtractPDF extracts text from a PDF byte slice safely (with panic recovery).
// Uses ledongthuc/pdf as primary engine and falls back to a pure-Go Flate + ToUnicode CMap
// stream extractor for complex/non-standard Chinese PDFs.
// Returns ErrEncryptedPDF for password-protected PDFs and ErrScannedPDF for image-only PDFs.
func ExtractPDF(data []byte) (text string, err error) {
	hasEncrypt := bytes.Contains(data, []byte("/Encrypt"))

	text, primaryErr := parsePDFSafe(data)
	if primaryErr == nil && strings.TrimSpace(text) != "" {
		return text, nil
	}

	// Try pure-Go Flate + ToUnicode CMap fallback when primary engine yields nothing or errors
	if !hasEncrypt {
		if fb := extractPDFFallback(data); strings.TrimSpace(fb) != "" {
			return fb, nil
		}
	}

	if hasEncrypt {
		return "", ErrEncryptedPDF
	}
	if primaryErr != nil {
		lower := strings.ToLower(primaryErr.Error())
		if strings.Contains(lower, "encrypt") || strings.Contains(lower, "password") {
			return "", ErrEncryptedPDF
		}
		return "", primaryErr
	}

	return "", ErrScannedPDF
}

func parsePDFSafe(data []byte) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("PDF parser panic recovered")
		}
	}()

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	var out strings.Builder
	numPages := reader.NumPage()
	for p := 1; p <= numPages; p++ {
		page := reader.Page(p)
		if page.V.IsNull() {
			continue
		}
		plain, err := page.GetPlainText(nil)
		if err == nil && plain != "" {
			out.WriteString(plain)
			out.WriteByte('\n')
		}
	}
	return out.String(), nil
}

// =========================================================================
// Pure-Go Fallback PDF Extractor: FlateDecode streams + ToUnicode CMap
// Handles non-standard / ObjStm / CJK ToUnicode PDFs when ledongthuc/pdf fails.
// =========================================================================

var (
	rePDFStream   = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\n?endstream`)
	reBfCharBlock = regexp.MustCompile(`(?s)beginbfchar(.*?)endbfchar`)
	reBfRangeBlk  = regexp.MustCompile(`(?s)beginbfrange(.*?)endbfrange`)
	reHexPair     = regexp.MustCompile(`<([0-9a-fA-F]+)>\s*<([0-9a-fA-F]+)>`)
	reHexTriple   = regexp.MustCompile(`<([0-9a-fA-F]+)>\s*<([0-9a-fA-F]+)>\s*<([0-9a-fA-F]+)>`)
)

func extractPDFFallback(data []byte) string {
	// Cap fallback scan size to prevent pathological memory usage
	const maxStreamOut = 4 * 1024 * 1024

	matches := rePDFStream.FindAllSubmatch(data, 500)
	if len(matches) == 0 {
		return ""
	}

	var decodedStreams [][]byte
	cmap := make(map[string]string) // uppercase hex code -> unicode string
	cmapCodeLens := map[int]bool{}
	totalDecoded := 0

	for _, m := range matches {
		if len(m) < 2 || len(m[1]) == 0 {
			continue
		}
		raw := m[1]
		var streamBytes []byte
		if dec, ok := tryFlateDecompress(raw, 2*1024*1024); ok {
			streamBytes = dec
		} else if !looksBinary(raw) {
			streamBytes = raw
		} else {
			continue
		}

		totalDecoded += len(streamBytes)
		if totalDecoded > maxStreamOut {
			break
		}

		if bytes.Contains(streamBytes, []byte("beginbfchar")) || bytes.Contains(streamBytes, []byte("beginbfrange")) {
			parseToUnicodeCMap(streamBytes, cmap, cmapCodeLens)
		} else if bytes.Contains(streamBytes, []byte("BT")) && (bytes.Contains(streamBytes, []byte("Tj")) || bytes.Contains(streamBytes, []byte("TJ"))) {
			decodedStreams = append(decodedStreams, streamBytes)
		}
	}

	if len(decodedStreams) == 0 {
		return ""
	}

	var out strings.Builder
	for _, st := range decodedStreams {
		txt := extractTextOperators(st, cmap, cmapCodeLens)
		if txt != "" {
			out.WriteString(txt)
			out.WriteByte('\n')
		}
	}
	return strings.TrimSpace(out.String())
}

func tryFlateDecompress(raw []byte, limit int64) ([]byte, bool) {
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, false
	}
	defer zr.Close()
	buf, err := io.ReadAll(io.LimitReader(zr, limit))
	if err != nil && len(buf) == 0 {
		return nil, false
	}
	return buf, len(buf) > 0
}

func parseToUnicodeCMap(stream []byte, cmap map[string]string, codeLens map[int]bool) {
	s := string(stream)
	for _, blk := range reBfCharBlock.FindAllStringSubmatch(s, -1) {
		for _, pair := range reHexPair.FindAllStringSubmatch(blk[1], -1) {
			src := strings.ToUpper(pair[1])
			dst := decodeUTF16BEHex(pair[2])
			if dst != "" {
				cmap[src] = dst
				codeLens[len(src)] = true
			}
		}
	}
	for _, blk := range reBfRangeBlk.FindAllStringSubmatch(s, -1) {
		for _, tri := range reHexTriple.FindAllStringSubmatch(blk[1], -1) {
			startVal, err1 := strconv.ParseUint(tri[1], 16, 32)
			endVal, err2 := strconv.ParseUint(tri[2], 16, 32)
			dstStart, err3 := strconv.ParseUint(tri[3], 16, 32)
			if err1 != nil || err2 != nil || err3 != nil || endVal < startVal || endVal-startVal > 4096 {
				continue
			}
			hexLen := len(tri[1])
			codeLens[hexLen] = true
			for cur := startVal; cur <= endVal; cur++ {
				key := strings.ToUpper(strconv.FormatUint(cur, 16))
				for len(key) < hexLen {
					key = "0" + key
				}
				r := rune(dstStart + (cur - startVal))
				cmap[key] = string(r)
			}
		}
	}
}

func decodeUTF16BEHex(h string) string {
	if len(h)%2 != 0 {
		h = "0" + h
	}
	b, err := hex.DecodeString(h)
	if err != nil || len(b) == 0 {
		return ""
	}
	if len(b) == 1 {
		return string(rune(b[0]))
	}
	if len(b)%2 != 0 {
		return string(b)
	}
	u16 := make([]uint16, len(b)/2)
	for i := 0; i < len(u16); i++ {
		u16[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
	}
	return string(utf16.Decode(u16))
}

func extractTextOperators(stream []byte, cmap map[string]string, codeLens map[int]bool) string {
	var out strings.Builder
	n := len(stream)
	i := 0
	for i < n {
		switch stream[i] {
		case '(':
			// Literal PDF string (...)
			var lit bytes.Buffer
			depth := 1
			i++
			for i < n && depth > 0 {
				c := stream[i]
				if c == '\\' && i+1 < n {
					i++
					switch stream[i] {
					case 'n':
						lit.WriteByte('\n')
					case 'r':
						lit.WriteByte('\r')
					case 't':
						lit.WriteByte('\t')
					case '(', ')', '\\':
						lit.WriteByte(stream[i])
					default:
						lit.WriteByte(stream[i])
					}
					i++
					continue
				}
				if c == '(' {
					depth++
				} else if c == ')' {
					depth--
					if depth == 0 {
						i++
						break
					}
				}
				lit.WriteByte(c)
				i++
			}
			// Only keep literal strings followed closely by Tj, ', ", or inside [...] TJ
			if isFollowedByTextOp(stream, i) {
				b := lit.Bytes()
				if len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF {
					out.WriteString(decodeUTF16BEHex(hex.EncodeToString(b[2:])))
				} else {
					out.WriteString(DecodeTextBytes(b))
				}
			}
		case '<':
			if i+1 < n && stream[i+1] == '<' {
				i += 2
				continue
			}
			end := bytes.IndexByte(stream[i+1:], '>')
			if end < 0 {
				i++
				continue
			}
			hexStr := strings.ToUpper(strings.Map(func(r rune) rune {
				if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'F') || (r >= 'a' && r <= 'f') {
					return r
				}
				return -1
			}, string(stream[i+1:i+1+end])))
			i = i + 2 + end
			if len(hexStr) > 0 && isFollowedByTextOp(stream, i) {
				if len(cmap) > 0 {
					out.WriteString(applyCMap(hexStr, cmap, codeLens))
				} else {
					out.WriteString(decodeUTF16BEHex(hexStr))
				}
			}
		default:
			i++
		}
	}
	return out.String()
}

func isFollowedByTextOp(stream []byte, pos int) bool {
	end := pos + 64
	if end > len(stream) {
		end = len(stream)
	}
	tail := stream[pos:end]
	return bytes.Contains(tail, []byte("Tj")) ||
		bytes.Contains(tail, []byte("TJ")) ||
		bytes.Contains(tail, []byte("'")) ||
		bytes.Contains(tail, []byte(`"`))
}

func applyCMap(hexStr string, cmap map[string]string, codeLens map[int]bool) string {
	var out strings.Builder
	// Prefer 4-hex-digit (2-byte CJK CID) before 2-hex-digit (1-byte)
	tryLens := []int{4, 2}
	for i := 0; i < len(hexStr); {
		matched := false
		for _, l := range tryLens {
			if codeLens[l] && i+l <= len(hexStr) {
				if val, ok := cmap[hexStr[i:i+l]]; ok {
					out.WriteString(val)
					i += l
					matched = true
					break
				}
			}
		}
		if !matched {
			if i+4 <= len(hexStr) && codeLens[4] {
				out.WriteString(decodeUTF16BEHex(hexStr[i : i+4]))
				i += 4
			} else if i+2 <= len(hexStr) {
				if b, err := hex.DecodeString(hexStr[i : i+2]); err == nil && len(b) == 1 && b[0] >= 0x20 && b[0] < 0x7f {
					out.WriteByte(b[0])
				}
				i += 2
			} else {
				i++
			}
		}
	}
	return out.String()
}
