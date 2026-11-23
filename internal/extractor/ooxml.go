package extractor

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

// ExtractOOXMLOrOFD inspects a ZIP-based office/document package (.docx, .xlsx, .pptx, .ofd, .odt, .ods, .odp)
// and extracts its textual content with zero external dependencies.
func ExtractOOXMLOrOFD(data []byte, hintExt string) (text string, detectedKind string, err error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", "", err
	}

	hasWord := false
	hasExcel := false
	hasPPT := false
	hasOFD := false
	hasODF := false

	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		switch {
		case name == "word/document.xml" || strings.HasPrefix(name, "word/"):
			hasWord = true
		case name == "xl/workbook.xml" || strings.HasPrefix(name, "xl/worksheets/"):
			hasExcel = true
		case strings.HasPrefix(name, "ppt/slides/"):
			hasPPT = true
		case name == "ofd.xml" || strings.HasSuffix(name, "/content.xml") && strings.Contains(name, "pages/"):
			hasOFD = true
		case name == "content.xml":
			hasODF = true
		}
	}

	switch {
	case hasWord:
		return extractDOCX(zr), "docx", nil
	case hasExcel:
		return extractXLSX(zr), "xlsx", nil
	case hasPPT:
		return extractPPTX(zr), "pptx", nil
	case hasOFD || hintExt == ".ofd":
		return extractOFD(zr), "ofd", nil
	case hasODF:
		return extractODF(zr), "odf", nil
	default:
		// Not an office document container — let archive handler inspect it if applicable
		return "", "zip", nil
	}
}

func extractDOCX(zr *zip.Reader) string {
	var out strings.Builder
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		if name == "word/document.xml" ||
			(strings.HasPrefix(name, "word/header") && strings.HasSuffix(name, ".xml")) ||
			(strings.HasPrefix(name, "word/footer") && strings.HasSuffix(name, ".xml")) ||
			name == "word/footnotes.xml" ||
			name == "word/endnotes.xml" ||
			name == "word/comments.xml" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			extractXMLTextNodes(rc, &out, map[string]bool{"t": true, "delText": true}, map[string]bool{"p": true, "tr": true, "br": true})
			_ = rc.Close()
			out.WriteByte('\n')
		}
	}
	return out.String()
}

func extractPPTX(zr *zip.Reader) string {
	var out strings.Builder
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		if (strings.HasPrefix(name, "ppt/slides/slide") ||
			strings.HasPrefix(name, "ppt/notesslides/notesslide") ||
			strings.HasPrefix(name, "ppt/comments/comment")) && strings.HasSuffix(name, ".xml") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			extractXMLTextNodes(rc, &out, map[string]bool{"t": true}, map[string]bool{"p": true, "br": true})
			_ = rc.Close()
			out.WriteByte('\n')
		}
	}
	return out.String()
}

func extractXLSX(zr *zip.Reader) string {
	// 1. Load sharedStrings.xml first
	var sharedStrings []string
	for _, f := range zr.File {
		if strings.ToLower(f.Name) == "xl/sharedstrings.xml" {
			rc, err := f.Open()
			if err == nil {
				sharedStrings = parseXLSXSharedStrings(rc)
				_ = rc.Close()
			}
			break
		}
	}

	var out strings.Builder
	// Emit shared strings directly as fallback if worksheets are unusually structured,
	// AND parse each sheet to preserve cell order and inline/numeric values.
	sheetFound := false
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		if strings.HasPrefix(name, "xl/worksheets/sheet") && strings.HasSuffix(name, ".xml") {
			sheetFound = true
			rc, err := f.Open()
			if err != nil {
				continue
			}
			parseXLSXSheet(rc, sharedStrings, &out)
			_ = rc.Close()
			out.WriteByte('\n')
		}
	}

	if !sheetFound && len(sharedStrings) > 0 {
		out.WriteString(strings.Join(sharedStrings, " "))
	}
	return out.String()
}

func parseXLSXSharedStrings(r io.Reader) []string {
	dec := xml.NewDecoder(r)
	var list []string
	var cur strings.Builder
	inSI := false
	inT := false

	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "si":
				inSI = true
				cur.Reset()
			case "t":
				if inSI {
					inT = true
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inT = false
			case "si":
				if inSI {
					list = append(list, cur.String())
					inSI = false
				}
			}
		case xml.CharData:
			if inSI && inT {
				cur.Write(t)
			}
		}
	}
	return list
}

func parseXLSXSheet(r io.Reader, sharedStrings []string, out *strings.Builder) {
	dec := xml.NewDecoder(r)
	cellType := ""
	inV := false
	inT := false
	var valBuf strings.Builder

	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "c":
				cellType = ""
				for _, attr := range t.Attr {
					if attr.Name.Local == "t" {
						cellType = attr.Value
						break
					}
				}
			case "v":
				inV = true
				valBuf.Reset()
			case "t":
				inT = true
				valBuf.Reset()
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "v":
				inV = false
				raw := strings.TrimSpace(valBuf.String())
				if raw != "" {
					if cellType == "s" {
						idx := atoiSafe(raw)
						if idx >= 0 && idx < len(sharedStrings) {
							out.WriteString(sharedStrings[idx])
							out.WriteByte(' ')
						}
					} else {
						out.WriteString(raw)
						out.WriteByte(' ')
					}
				}
			case "t":
				inT = false
				raw := valBuf.String()
				if raw != "" {
					out.WriteString(raw)
					out.WriteByte(' ')
				}
			case "row":
				out.WriteByte('\n')
			}
		case xml.CharData:
			if inV || inT {
				valBuf.Write(t)
			}
		}
	}
}

// extractOFD extracts text from Chinese national standard OFD (GB/T 33190) documents.
func extractOFD(zr *zip.Reader) string {
	var out strings.Builder
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		if strings.HasSuffix(name, ".xml") &&
			(strings.Contains(name, "content") || strings.Contains(name, "page") || strings.Contains(name, "annot")) {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			extractXMLTextNodes(rc, &out, map[string]bool{"TextCode": true, "textcode": true}, map[string]bool{"TextObject": true, "Page": true})
			_ = rc.Close()
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// extractODF extracts text from OpenDocument formats (.odt, .ods, .odp).
func extractODF(zr *zip.Reader) string {
	var out strings.Builder
	for _, f := range zr.File {
		if strings.ToLower(f.Name) == "content.xml" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			dec := xml.NewDecoder(rc)
			for {
				tok, err := dec.Token()
				if err != nil {
					break
				}
				switch t := tok.(type) {
				case xml.CharData:
					out.Write(t)
				case xml.EndElement:
					if t.Name.Local == "p" || t.Name.Local == "h" || t.Name.Local == "table-row" {
						out.WriteByte('\n')
					} else if t.Name.Local == "table-cell" {
						out.WriteByte(' ')
					}
				}
			}
			_ = rc.Close()
		}
	}
	return out.String()
}

func extractXMLTextNodes(r io.Reader, out *strings.Builder, textTags map[string]bool, breakTags map[string]bool) {
	dec := xml.NewDecoder(r)
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if textTags[t.Name.Local] || textTags[strings.ToLower(t.Name.Local)] {
				depth++
			}
		case xml.EndElement:
			if textTags[t.Name.Local] || textTags[strings.ToLower(t.Name.Local)] {
				if depth > 0 {
					depth--
				}
			}
			if breakTags[t.Name.Local] {
				out.WriteByte('\n')
			}
		case xml.CharData:
			if depth > 0 {
				out.Write(t)
			}
		}
	}
}

func atoiSafe(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
	}
	return n
}
