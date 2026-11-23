package extractor

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestDecodeTextBytesGBKAndUTF8(t *testing.T) {
	gbkBytes, err := simplifiedchinese.GB18030.NewEncoder().Bytes([]byte("银河麒麟系统公文归档测试"))
	if err != nil {
		t.Fatal(err)
	}
	got := DecodeTextBytes(gbkBytes)
	if got != "银河麒麟系统公文归档测试" {
		t.Fatalf("GBK decode failed: got %q", got)
	}
}

func TestFakeDocRTFAndHTML(t *testing.T) {
	// Fake .doc that is actually RTF with Chinese \u escapes & GBK \'xx
	rtf := `{\rtf1\ansi\ansicpg936{\fonttbl{\f0 SimSun;}}\u20445?\u23494?\par \'c4\'ea\'b6\'c8\'d4\'a4\'cb\'e3}`
	txt, skip, err := ExtractBytes([]byte(rtf), "合同.doc")
	if err != nil || skip != "" {
		t.Fatalf("unexpected error/skip: %v / %s", err, skip)
	}
	if !strings.Contains(txt, "保密") || !strings.Contains(txt, "年度预算") {
		t.Fatalf("RTF Chinese extraction failed: %q", txt)
	}

	// Fake .doc that is actually HTML exported by OA system
	htmlDoc := `<!DOCTYPE html><html><head><style>body{color:red;}</style></head><body><h1>机密通知</h1><p>请于周五前清理。</p></body></html>`
	txt2, skip2, err2 := ExtractBytes([]byte(htmlDoc), "通知.doc")
	if err2 != nil || skip2 != "" {
		t.Fatalf("unexpected error/skip: %v / %s", err2, skip2)
	}
	if !strings.Contains(txt2, "机密通知") || strings.Contains(txt2, "color:red") {
		t.Fatalf("HTML fake .doc extraction failed: %q", txt2)
	}
}

func TestOOXMLAndOFD(t *testing.T) {
	// Build minimal DOCX in memory
	var docxBuf bytes.Buffer
	zw := zip.NewWriter(&docxBuf)
	f, _ := zw.Create("word/document.xml")
	_, _ = f.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>涉密项目总结报告</w:t></w:r></w:p></w:body></w:document>`))
	_ = zw.Close()

	txt, skip, err := ExtractBytes(docxBuf.Bytes(), "report.docx")
	if err != nil || skip != "" || !strings.Contains(txt, "涉密项目总结报告") {
		t.Fatalf("DOCX extraction failed: txt=%q skip=%q err=%v", txt, skip, err)
	}

	// Build minimal OFD in memory
	var ofdBuf bytes.Buffer
	zw2 := zip.NewWriter(&ofdBuf)
	f1, _ := zw2.Create("OFD.xml")
	_, _ = f1.Write([]byte(`<ofd:OFD xmlns:ofd="http://www.ofdspec.org/2016"></ofd:OFD>`))
	f2, _ := zw2.Create("Doc_0/Pages/Page_0/Content.xml")
	_, _ = f2.Write([]byte(`<ofd:Page xmlns:ofd="http://www.ofdspec.org/2016"><ofd:Content><ofd:Layer><ofd:TextObject><ofd:TextCode>红头文件审批单</ofd:TextCode></ofd:TextObject></ofd:Layer></ofd:Content></ofd:Page>`))
	_ = zw2.Close()

	txtOFD, skipOFD, errOFD := ExtractBytes(ofdBuf.Bytes(), "公文.ofd")
	if errOFD != nil || skipOFD != "" || !strings.Contains(txtOFD, "红头文件审批单") {
		t.Fatalf("OFD extraction failed: txt=%q skip=%q err=%v", txtOFD, skipOFD, errOFD)
	}
}

func TestPPTStreamRecordParsing(t *testing.T) {
	// Construct a minimal PPT TextCharsAtom (0x0FA0) inside a container (ver=0x0F)
	msg := "季度经营复盘PPT"
	var payload bytes.Buffer
	for _, r := range msg {
		_ = binary.Write(&payload, binary.LittleEndian, uint16(r))
	}
	var atom bytes.Buffer
	_ = binary.Write(&atom, binary.LittleEndian, uint16(0x0000)) // ver/inst
	_ = binary.Write(&atom, binary.LittleEndian, uint16(0x0FA0)) // TextCharsAtom
	_ = binary.Write(&atom, binary.LittleEndian, uint32(payload.Len()))
	atom.Write(payload.Bytes())

	got := extractPPTStream(atom.Bytes())
	if !strings.Contains(got, msg) {
		t.Fatalf("PPT stream parse failed: got %q", got)
	}
}

func TestWalkZipGBKFilenameAndContent(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "archive.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	gbkName, _ := simplifiedchinese.GB18030.NewEncoder().String("财务/机密清单.txt")
	hdr := &zip.FileHeader{
		Name:   gbkName,
		Method: zip.Deflate,
		Flags:  0, // Non-UTF8 Windows zip
	}
	w, _ := zw.CreateHeader(hdr)
	_, _ = w.Write([]byte("包含内部合同金额数据"))
	_ = zw.Close()
	_ = f.Close()

	var found []ArchiveItem
	skip := WalkArchiveFile(zipPath, 2, func(item ArchiveItem) {
		found = append(found, item)
	})
	if skip != "" || len(found) != 1 {
		t.Fatalf("expected 1 archive item, got %d (skip=%q)", len(found), skip)
	}
	if found[0].FileName != "机密清单.txt" || !strings.Contains(found[0].InnerPath, "财务/机密清单.txt") {
		t.Fatalf("GBK zip filename decoding failed: %+v", found[0])
	}
	txt, _, _ := ExtractBytes(found[0].Data, found[0].FileName)
	if !strings.Contains(txt, "内部合同金额") {
		t.Fatalf("zip inner content extraction failed: %q", txt)
	}
}
