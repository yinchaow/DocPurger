package extractor

import (
	"bytes"
	"errors"
	"strings"

	"github.com/ledongthuc/pdf"
)

var (
	ErrEncryptedPDF = errors.New("encrypted PDF")
	ErrScannedPDF   = errors.New("scanned or image-only PDF without text layer")
)

// ExtractPDF extracts text from a PDF byte slice safely (with panic recovery).
// Returns ErrEncryptedPDF for password-protected PDFs and ErrScannedPDF for image-only PDFs.
func ExtractPDF(data []byte) (text string, err error) {
	// Quick heuristic for encrypted PDFs
	if bytes.Contains(data, []byte("/Encrypt")) {
		// Some PDFs have empty/default user password; still try parsing first, then report encrypted if it fails
		text, err = parsePDFSafe(data)
		if err != nil || strings.TrimSpace(text) == "" {
			return "", ErrEncryptedPDF
		}
		return text, nil
	}

	text, err = parsePDFSafe(data)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "encrypt") || strings.Contains(lower, "password") {
			return "", ErrEncryptedPDF
		}
		return "", err
	}

	if strings.TrimSpace(text) == "" {
		return "", ErrScannedPDF
	}
	return text, nil
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
