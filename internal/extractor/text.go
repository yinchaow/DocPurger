package extractor

import (
	"bytes"
	"encoding/binary"
	"html"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// DecodeTextBytes detects UTF-8 / UTF-16LE / UTF-16BE / GB18030 and returns clean UTF-8 text.
func DecodeTextBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// 1. Check BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return string(data[3:])
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		return decodeUTF16(data[2:], binary.LittleEndian)
	}
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return decodeUTF16(data[2:], binary.BigEndian)
	}

	// 2. Check heuristic UTF-16 without BOM (lots of zero bytes at odd/even positions)
	if len(data) >= 4 && len(data)%2 == 0 {
		sample := data
		if len(sample) > 512 {
			sample = sample[:512]
		}
		evenZeros, oddZeros := 0, 0
		for i := 0; i < len(sample); i += 2 {
			if sample[i] == 0 {
				evenZeros++
			}
			if sample[i+1] == 0 {
				oddZeros++
			}
		}
		half := len(sample) / 2
		if oddZeros > half/3 && evenZeros < half/8 {
			return decodeUTF16(data, binary.LittleEndian)
		}
		if evenZeros > half/3 && oddZeros < half/8 {
			return decodeUTF16(data, binary.BigEndian)
		}
	}

	// 3. Valid UTF-8
	if utf8.Valid(data) {
		return string(data)
	}

	// 4. Fallback to GB18030 (superset of GBK / GB2312, covers Windows Chinese text files)
	decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(data)
	if err == nil {
		return string(decoded)
	}
	return string(data)
}

func decodeUTF16(data []byte, order binary.ByteOrder) string {
	n := len(data) / 2
	u16 := make([]uint16, n)
	for i := 0; i < n; i++ {
		u16[i] = order.Uint16(data[i*2 : i*2+2])
	}
	return string(utf16.Decode(u16))
}

// DecodeGBKString converts raw bytes (e.g. non-UTF8 ZIP filenames) to UTF-8.
func DecodeGBKString(raw string) string {
	if utf8.ValidString(raw) {
		return raw
	}
	decoded, err := simplifiedchinese.GB18030.NewDecoder().String(raw)
	if err != nil {
		return raw
	}
	return decoded
}

// ExtractHTMLText strips scripts, styles, and HTML/XML tags and unescapes HTML entities.
// Handles HTML/MHT files that are frequently saved with a .doc or .xls extension in China.
func ExtractHTMLText(data []byte) string {
	raw := DecodeTextBytes(data)
	if raw == "" {
		return ""
	}

	// If MHT / MIME, strip quoted-printable soft line breaks
	lowerHead := strings.ToLower(raw)
	if len(lowerHead) > 1024 {
		lowerHead = lowerHead[:1024]
	}
	if strings.Contains(lowerHead, "content-transfer-encoding: quoted-printable") {
		raw = decodeQuotedPrintable(raw)
	}

	// Strip <script> and <style> blocks
	raw = stripTagBlock(raw, "script")
	raw = stripTagBlock(raw, "style")
	raw = stripTagBlock(raw, "head")

	var out strings.Builder
	out.Grow(len(raw) / 2)
	inTag := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c == '<' {
			inTag = true
			out.WriteByte(' ')
			continue
		}
		if c == '>' {
			inTag = false
			continue
		}
		if !inTag {
			out.WriteByte(c)
		}
	}

	return html.UnescapeString(out.String())
}

func stripTagBlock(s, tag string) string {
	lower := strings.ToLower(s)
	openTag := "<" + tag
	closeTag := "</" + tag + ">"
	var b strings.Builder
	b.Grow(len(s))

	pos := 0
	for {
		idx := strings.Index(lower[pos:], openTag)
		if idx < 0 {
			b.WriteString(s[pos:])
			break
		}
		b.WriteString(s[pos : pos+idx])
		endIdx := strings.Index(lower[pos+idx:], closeTag)
		if endIdx < 0 {
			break
		}
		pos = pos + idx + endIdx + len(closeTag)
	}
	return b.String()
}

func decodeQuotedPrintable(s string) string {
	s = strings.ReplaceAll(s, "=\r\n", "")
	s = strings.ReplaceAll(s, "=\n", "")
	var out bytes.Buffer
	out.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '=' && i+2 < len(s) {
			if v, err := strconv.ParseUint(s[i+1:i+3], 16, 8); err == nil {
				out.WriteByte(byte(v))
				i += 2
				continue
			}
		}
		out.WriteByte(s[i])
	}
	return DecodeTextBytes(out.Bytes())
}

// ExtractRTFText parses RTF documents (including Chinese \'xx GBK hex escapes and \uNNNN Unicode escapes).
// Many legacy Chinese ".doc" files are actually RTF documents with a .doc extension!
func ExtractRTFText(data []byte) string {
	var out strings.Builder
	out.Grow(len(data) / 2)

	n := len(data)
	skipDepth := 0
	depth := 0
	ucSkip := 1 // Default number of ANSI fallback chars after \uN

	// Buffer consecutive \'xx bytes so multi-byte GBK characters decode cleanly
	var hexBuf []byte
	flushHex := func() {
		if len(hexBuf) == 0 {
			return
		}
		decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(hexBuf)
		if err == nil {
			out.Write(decoded)
		} else {
			out.Write(hexBuf)
		}
		hexBuf = hexBuf[:0]
	}

	skipDestinations := map[string]bool{
		"fonttbl": true, "colortbl": true, "stylesheet": true, "info": true,
		"pict": true, "object": true, "header": true, "footer": true,
		"listtable": true, "listoverridetable": true, "revtbl": true,
		"rsidtbl": true, "themedata": true, "colorschememapping": true,
		"datastore": true, "latentstyles": true, "xmlnstbl": true,
		"mmathPr": true, "fldinst": true, "bkmkstart": true, "bkmkend": true,
	}

	for i := 0; i < n; i++ {
		c := data[i]

		if c == '{' {
			flushHex()
			depth++
			// Check for ignorable destination {\*...
			if i+2 < n && data[i+1] == '\\' && data[i+2] == '*' && skipDepth == 0 {
				skipDepth = depth
			}
			continue
		}
		if c == '}' {
			flushHex()
			if skipDepth == depth {
				skipDepth = 0
			}
			if depth > 0 {
				depth--
			}
			continue
		}
		if skipDepth > 0 {
			continue
		}
		if c == '\r' || c == '\n' {
			continue
		}

		if c == '\\' {
			if i+1 >= n {
				break
			}
			next := data[i+1]

			// Hex escape \'xx (GBK / ANSI bytes)
			if next == '\'' && i+3 < n {
				if v, err := strconv.ParseUint(string(data[i+2:i+4]), 16, 8); err == nil {
					hexBuf = append(hexBuf, byte(v))
					i += 3
					continue
				}
			}

			flushHex()

			// Escaped literal characters
			if next == '\\' || next == '{' || next == '}' {
				out.WriteByte(next)
				i++
				continue
			}
			if next == '~' {
				out.WriteByte(' ')
				i++
				continue
			}
			if next == '\r' || next == '\n' {
				out.WriteByte('\n')
				i++
				continue
			}
			if next == '*' {
				skipDepth = depth
				i++
				continue
			}

			// Parse control word + optional signed numeric parameter
			j := i + 1
			for j < n && ((data[j] >= 'a' && data[j] <= 'z') || (data[j] >= 'A' && data[j] <= 'Z')) {
				j++
			}
			word := string(data[i+1 : j])

			k := j
			if k < n && (data[k] == '-' || (data[k] >= '0' && data[k] <= '9')) {
				if data[k] == '-' {
					k++
				}
				for k < n && data[k] >= '0' && data[k] <= '9' {
					k++
				}
			}
			paramStr := string(data[j:k])
			if k < n && data[k] == ' ' {
				k++ // Consume single delimiter space
			}
			i = k - 1

			if skipDestinations[word] {
				skipDepth = depth
				continue
			}

			switch word {
			case "par", "line", "row":
				out.WriteByte('\n')
			case "tab", "cell":
				out.WriteByte(' ')
			case "uc":
				if v, err := strconv.Atoi(paramStr); err == nil && v >= 0 {
					ucSkip = v
				}
			case "u":
				if v, err := strconv.Atoi(paramStr); err == nil {
					if v < 0 {
						v += 65536
					}
					if v > 0 && v <= 0x10FFFF {
						out.WriteRune(rune(v))
					}
					// Skip replacement ANSI chars following \uN
					skipped := 0
					for skipped < ucSkip && i+1 < n {
						if data[i+1] == '\\' && i+4 < n && data[i+2] == '\'' {
							i += 4
						} else if data[i+1] == '{' || data[i+1] == '}' {
							break
						} else {
							i++
						}
						skipped++
					}
				}
			}
			continue
		}

		flushHex()
		out.WriteByte(c)
	}
	flushHex()
	return out.String()
}
