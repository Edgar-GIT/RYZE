package document_screening

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func pngWithText(t *testing.T, key, value string, size int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	if key == "" {
		return data
	}
	// Insert a tEXt chunk right after IHDR (8-byte signature + 25-byte IHDR).
	payload := append([]byte(key+"\x00"), []byte(value)...)
	chunk := make([]byte, 0, len(payload)+12)
	chunk = binary.BigEndian.AppendUint32(chunk, uint32(len(payload)))
	typed := append([]byte("tEXt"), payload...)
	chunk = append(chunk, typed...)
	chunk = binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(typed))
	out := append([]byte{}, data[:33]...)
	out = append(out, chunk...)
	return append(out, data[33:]...)
}

func TestRejectsAIGeneratedImages(t *testing.T) {
	cases := map[string][]byte{
		"a1111 parameters": pngWithText(t, "parameters", "a certificate, Steps: 30, Sampler: Euler", 800),
		"comfyui workflow": pngWithText(t, "workflow", `{"nodes":[]}`, 800),
		"software name":    pngWithText(t, "Software", "Midjourney v6", 800),
		"iptc source type": append(pngWithText(t, "", "", 800), []byte("Iptc4xmpExt:DigitalSourceType=http://cv.iptc.org/newscodes/digitalsourcetype/trainedAlgorithmicMedia")...),
		"c2pa manifest":    append(pngWithText(t, "", "", 800), []byte("caBX....jumb....c2pa.claim....claim_generator: DALL-E 3 / OpenAI")...),
	}
	for name, data := range cases {
		if verdict := Screen("image/png", data); !verdict.Rejected {
			t.Fatalf("%s: expected a rejection, got %+v", name, verdict)
		}
	}
}

func TestPassesAPlainScanButSignalsReviewHints(t *testing.T) {
	verdict := Screen("image/png", pngWithText(t, "", "", 800))
	if verdict.Rejected {
		t.Fatalf("a plain image must not be rejected: %+v", verdict)
	}
	if verdict.Status() != "flagged" || !contains(verdict.Signals, SignalNoCameraMetadata) {
		t.Fatalf("expected the no-camera hint for the reviewer: %+v", verdict)
	}
	small := Screen("image/png", pngWithText(t, "", "", 200))
	if !contains(small.Signals, SignalLowResolution) {
		t.Fatalf("expected the low-resolution hint: %+v", small)
	}
}

func TestPDFChecks(t *testing.T) {
	clean := []byte("%PDF-1.7\n1 0 obj << /Producer (Microsoft Word) /CreationDate (D:20240101120000) /ModDate (D:20240101120000) >>\nendobj\n" +
		"2 0 obj << /Length 9 >> stream\n/JS (x)\x00\nendstream endobj\n%%EOF\n")
	if verdict := Screen("application/pdf", clean); verdict.Rejected || len(verdict.Signals) != 0 {
		t.Fatalf("a clean PDF (with /JS-like bytes inside a stream) must pass: %+v", verdict)
	}
	script := []byte("%PDF-1.7\n1 0 obj << /OpenAction << /S /JavaScript /JS (app.alert(1)) >> >>\nendobj\n%%EOF\n")
	if verdict := Screen("application/pdf", script); !verdict.Rejected || !contains(verdict.Reasons, ReasonUnsafePDF) {
		t.Fatalf("a PDF with a script must be rejected: %+v", verdict)
	}
	generated := []byte("%PDF-1.7\n1 0 obj << /Producer (ChatGPT export) >>\nendobj\n%%EOF\n")
	if verdict := Screen("application/pdf", generated); !contains(verdict.Reasons, ReasonAIGenerator) {
		t.Fatalf("a PDF produced by an AI tool must be rejected: %+v", verdict)
	}
	edited := []byte("%PDF-1.7\n1 0 obj << /Producer (iLovePDF) /CreationDate (D:20230101) /ModDate (D:20240505) >>\nendobj\n%%EOF\n2 0 obj<<>>endobj\n%%EOF\n")
	verdict := Screen("application/pdf", edited)
	if verdict.Rejected || !contains(verdict.Signals, SignalEditedWithEditor) || !contains(verdict.Signals, SignalModifiedAfterCreation) || !contains(verdict.Signals, SignalIncrementalEdits) {
		t.Fatalf("an edited PDF is flagged for review, not rejected: %+v", verdict)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
