package extractor

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/richardlehane/mscfb"
	"github.com/shakinm/xlsReader/xls"
	"golang.org/x/text/encoding/simplifiedchinese"
)

var ErrEncryptedDocument = errors.New("encrypted document")

// ExtractOLE2 parses legacy Microsoft Office OLE2/CFB binary formats (.doc, .xls, .ppt, .wps).
func ExtractOLE2(data []byte, hintExt string) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("malformed legacy office document")
		}
	}()

	reader, err := mscfb.New(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	streams := make(map[string][]byte)
	for entry, err := reader.Next(); err == nil; entry, err = reader.Next() {
		if entry.Size <= 0 || entry.Size > 64*1024*1024 {
			continue
		}
		buf := make([]byte, entry.Size)
		n, _ := io.ReadFull(reader, buf)
		streams[entry.Name] = buf[:n]
	}

	// Route by actual internal OLE2 stream names (more accurate than file extension!)
	if _, ok := streams["Workbook"]; ok {
		return extractXLS(data)
	}
	if _, ok := streams["Book"]; ok {
		return extractXLS(data)
	}
	if pptStream, ok := streams["PowerPoint Document"]; ok {
		return extractPPTStream(pptStream), nil
	}
	if wordDoc, ok := streams["WordDocument"]; ok {
		return extractDOCStreams(wordDoc, streams)
	}

	// Fallback by extension or generic stream text extraction (e.g. legacy .wps)
	if hintExt == ".xls" {
		return extractXLS(data)
	}
	var fallback strings.Builder
	for _, buf := range streams {
		s := extractFallbackReadableText(buf)
		if s != "" {
			fallback.WriteString(s)
			fallback.WriteByte('\n')
		}
	}
	return fallback.String(), nil
}

// extractXLS parses legacy .xls workbooks via pure-Go xlsReader.
func extractXLS(data []byte) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("failed to parse xls")
		}
	}()

	wb, err := xls.OpenReader(bytes.NewReader(data))
	if err != nil {
		// Check if encryption error
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "encrypt") || strings.Contains(lower, "password") {
			return "", ErrEncryptedDocument
		}
		return "", err
	}

	var out strings.Builder
	numSheets := wb.GetNumberSheets()
	for s := 0; s < numSheets; s++ {
		sheet, err := wb.GetSheet(s)
		if err != nil || sheet == nil {
			continue
		}
		if name := sheet.GetName(); name != "" {
			out.WriteString(name)
			out.WriteByte('\n')
		}
		rows := sheet.GetNumberRows()
		for r := 0; r < rows; r++ {
			row, err := sheet.GetRow(r)
			if err != nil || row == nil {
				continue
			}
			cols := row.GetCols()
			for _, col := range cols {
				if col == nil {
					continue
				}
				val := strings.TrimSpace(col.GetString())
				if val != "" {
					out.WriteString(val)
					out.WriteByte(' ')
				}
			}
			out.WriteByte('\n')
		}
	}
	return out.String(), nil
}

// extractDOCStreams parses Word Binary Format (.doc) using FIB + Clx/PieceTable (MS-DOC spec).
// Falls back to heuristic UTF-16LE/GB18030 text scanning if PieceTable is non-standard.
func extractDOCStreams(wordDoc []byte, streams map[string][]byte) (string, error) {
	if len(wordDoc) < 0x1A6 {
		return extractFallbackReadableText(wordDoc), nil
	}

	// Verify Word FIB magic (wIdent == 0xA5EC)
	wIdent := binary.LittleEndian.Uint16(wordDoc[0:2])
	if wIdent != 0xA5EC {
		return extractFallbackReadableText(wordDoc), nil
	}

	// Check flags at offset 0x0A (bytes 10..11)
	flags := binary.LittleEndian.Uint16(wordDoc[10:12])
	fEncrypted := (flags & 0x0100) != 0
	if fEncrypted {
		return "", ErrEncryptedDocument
	}
	fWhichTblStm := (flags & 0x0200) != 0

	tableName := "0Table"
	if fWhichTblStm {
		tableName = "1Table"
	}
	tableStream, hasTable := streams[tableName]
	if !hasTable {
		// Try the alternate table stream just in case
		if t, ok := streams["1Table"]; ok {
			tableStream = t
			hasTable = true
		} else if t, ok := streams["0Table"]; ok {
			tableStream = t
			hasTable = true
		}
	}

	if hasTable {
		if text, ok := parseDOCPieceTable(wordDoc, tableStream); ok && strings.TrimSpace(text) != "" {
			return text, nil
		}
	}

	// Fallback: extract text directly from WordDocument using FIB fcMin..fcMac or heuristic scan
	if len(wordDoc) >= 0x20 {
		fcMin := int(binary.LittleEndian.Uint32(wordDoc[0x18:0x1C]))
		fcMac := int(binary.LittleEndian.Uint32(wordDoc[0x1C:0x20]))
		if fcMin >= 0 && fcMac > fcMin && fcMac <= len(wordDoc) {
			if s := extractFallbackReadableText(wordDoc[fcMin:fcMac]); s != "" {
				return s, nil
			}
		}
	}
	return extractFallbackReadableText(wordDoc), nil
}

// parseDOCPieceTable locates Clx (fcClx/lcbClx at FIB offset 0x01A2) and reconstructs text pieces.
func parseDOCPieceTable(wordDoc, tableStream []byte) (string, bool) {
	if len(wordDoc) < 0x01AA {
		return "", false
	}
	fcClx := int(binary.LittleEndian.Uint32(wordDoc[0x01A2:0x01A6]))
	lcbClx := int(binary.LittleEndian.Uint32(wordDoc[0x01A6:0x01AA]))
	if fcClx < 0 || lcbClx <= 0 || fcClx+lcbClx > len(tableStream) {
		return "", false
	}

	clx := tableStream[fcClx : fcClx+lcbClx]
	pos := 0

	// Skip Prc records (clxt == 0x01)
	for pos < len(clx) && clx[pos] == 0x01 {
		if pos+3 > len(clx) {
			return "", false
		}
		cbGrpprl := int(binary.LittleEndian.Uint16(clx[pos+1 : pos+3]))
		pos += 3 + cbGrpprl
	}

	// Expect Pcdt record (clxt == 0x02)
	if pos+5 > len(clx) || clx[pos] != 0x02 {
		return "", false
	}
	lcb := int(binary.LittleEndian.Uint32(clx[pos+1 : pos+5]))
	pos += 5
	if lcb <= 4 || pos+lcb > len(clx) {
		return "", false
	}

	plcPcd := clx[pos : pos+lcb]
	// PlcPcd layout: (n+1) uint32 CPs followed by n 8-byte PCDs => 4*(n+1) + 8*n = 12*n + 4
	if (lcb-4)%12 != 0 {
		return "", false
	}
	n := (lcb - 4) / 12
	if n <= 0 || n > 500000 {
		return "", false
	}

	cpBase := 0
	pcdBase := (n + 1) * 4

	var out strings.Builder
	for i := 0; i < n; i++ {
		cpStart := binary.LittleEndian.Uint32(plcPcd[cpBase+i*4 : cpBase+(i+1)*4])
		cpEnd := binary.LittleEndian.Uint32(plcPcd[cpBase+(i+1)*4 : cpBase+(i+2)*4])
		if cpEnd <= cpStart {
			continue
		}
		charCount := int(cpEnd - cpStart)
		if charCount > 10*1024*1024 {
			continue
		}

		// PCD is 8 bytes: 2 bytes flags, 4 bytes fcCompressed, 2 bytes prm
		pcdOffset := pcdBase + i*8
		fcRaw := binary.LittleEndian.Uint32(plcPcd[pcdOffset+2 : pcdOffset+6])

		isCompressed := (fcRaw & 0x40000000) != 0
		fc := int(fcRaw & 0x3FFFFFFF)

		if isCompressed {
			byteStart := fc / 2
			byteEnd := byteStart + charCount
			if byteStart < 0 || byteEnd > len(wordDoc) {
				continue
			}
			chunk := wordDoc[byteStart:byteEnd]
			writeCleanWordChunk(&out, chunk, true)
		} else {
			byteStart := fc
			byteEnd := byteStart + charCount*2
			if byteStart < 0 || byteEnd > len(wordDoc) {
				continue
			}
			chunk := wordDoc[byteStart:byteEnd]
			writeCleanWordChunk(&out, chunk, false)
		}
	}

	return out.String(), true
}

func writeCleanWordChunk(out *strings.Builder, chunk []byte, compressed bool) {
	var text string
	if compressed {
		// Try GB18030 for legacy Chinese ANSI pieces; pure ASCII passes through untouched
		if decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(chunk); err == nil {
			text = string(decoded)
		} else {
			text = string(chunk)
		}
	} else {
		n := len(chunk) / 2
		u16 := make([]uint16, n)
		for i := 0; i < n; i++ {
			u16[i] = binary.LittleEndian.Uint16(chunk[i*2 : i*2+2])
		}
		text = string(utf16.Decode(u16))
	}

	// Clean Word special control characters (field markers 0x13/0x14/0x15, cell mark 0x07, etc.)
	inFieldCode := false
	for _, r := range text {
		switch r {
		case 0x13: // Field begin
			inFieldCode = true
		case 0x14: // Field separator (display text follows)
			inFieldCode = false
		case 0x15: // Field end
			inFieldCode = false
		case '\r', 0x0B, 0x0C:
			if !inFieldCode {
				out.WriteByte('\n')
			}
		case '\t', 0x07:
			if !inFieldCode {
				out.WriteByte(' ')
			}
		default:
			if !inFieldCode && r >= 0x20 && r != 0xFFFD {
				out.WriteRune(r)
			}
		}
	}
}

// extractPPTStream walks PowerPoint Binary Format records and extracts TextCharsAtom (0x0FA0) & TextBytesAtom (0x0FA8).
func extractPPTStream(stream []byte) string {
	var out strings.Builder
	walkPPTRecords(stream, 0, len(stream), &out, 0)
	return out.String()
}

func walkPPTRecords(buf []byte, start, end int, out *strings.Builder, depth int) {
	if depth > 16 {
		return
	}
	pos := start
	for pos+8 <= end {
		verInst := binary.LittleEndian.Uint16(buf[pos : pos+2])
		recType := binary.LittleEndian.Uint16(buf[pos+2 : pos+4])
		recLen := int(binary.LittleEndian.Uint32(buf[pos+4 : pos+8]))
		pos += 8

		if recLen < 0 || pos+recLen > end {
			break
		}

		recVer := verInst & 0x000F
		if recVer == 0x000F {
			// Container record — recurse
			walkPPTRecords(buf, pos, pos+recLen, out, depth+1)
		} else {
			switch recType {
			case 0x0FA0: // TextCharsAtom (UTF-16LE)
				s := decodeUTF16(buf[pos:pos+recLen], binary.LittleEndian)
				out.WriteString(strings.ReplaceAll(s, "\r", "\n"))
				out.WriteByte('\n')
			case 0x0FA8: // TextBytesAtom (ANSI / GB18030)
				if decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(buf[pos : pos+recLen]); err == nil {
					out.WriteString(strings.ReplaceAll(string(decoded), "\r", "\n"))
				} else {
					out.WriteString(strings.ReplaceAll(string(buf[pos:pos+recLen]), "\r", "\n"))
				}
				out.WriteByte('\n')
			}
		}
		pos += recLen
	}
}

// extractFallbackReadableText scans a binary stream for readable UTF-16LE and ASCII/GBK runs.
func extractFallbackReadableText(data []byte) string {
	var out strings.Builder

	// Scan UTF-16LE runs (captures Chinese & English stored in OLE2 Word/WPS streams)
	var cur []rune
	flush := func() {
		if len(cur) >= 4 {
			out.WriteString(string(cur))
			out.WriteByte(' ')
		}
		cur = cur[:0]
	}

	for i := 0; i+1 < len(data); i += 2 {
		v := binary.LittleEndian.Uint16(data[i : i+2])
		r := rune(v)
		if isReadableDocRune(r) {
			cur = append(cur, r)
		} else {
			flush()
		}
	}
	flush()
	return out.String()
}

func isReadableDocRune(r rune) bool {
	if r == '\n' || r == '\t' || r == ' ' {
		return true
	}
	if r >= 0x20 && r <= 0x7E {
		return true
	}
	if unicode.Is(unicode.Han, r) {
		return true
	}
	// Common CJK punctuation & fullwidth characters
	if (r >= 0x3000 && r <= 0x303F) || (r >= 0xFF00 && r <= 0xFFEF) {
		return true
	}
	return false
}
