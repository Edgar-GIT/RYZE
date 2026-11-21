// Package document_screening checks trainer evidence files before they are
// accepted (owner request, 2026-11-20: no fake entities on the platform).
//
// It looks for traces that a file was made or altered by generative AI —
// content credentials (C2PA) and IPTC "digital source type" values declaring
// synthetic media, AI generators named in the image or PDF metadata, prompts
// written into PNG text chunks — and for active or unsafe PDF content. Those
// files are rejected. Weaker signs (edited after creation, edited with an
// image editor, low resolution, no camera data) do not reject a file; they
// are recorded for the human review that every application goes through.
//
// No automatic check can prove a document is genuine: a generated image with
// its metadata removed carries no trace. The screening removes the obvious
// cases and gives the reviewer the signals; the decision stays human.
package document_screening

import (
	"bytes"
	"image"
	_ "image/jpeg" // registers the JPEG decoder for dimension checks
	_ "image/png"  // registers the PNG decoder for dimension checks
	"regexp"
	"strings"
)

// Rejection reasons (shown to the applicant).
const (
	ReasonAIProvenance = "ai_provenance"         // content credentials / IPTC declare synthetic media
	ReasonAIGenerator  = "ai_generator_metadata" // an AI generator named in the metadata or a prompt embedded
	ReasonUnsafePDF    = "unsafe_pdf_content"    // scripts, launch actions or embedded files in a PDF
)

// Review signals (shown to the administrator only).
const (
	SignalModifiedAfterCreation = "modified_after_creation"
	SignalIncrementalEdits      = "incremental_edits"
	SignalEditedWithEditor      = "edited_with_editor"
	SignalLowResolution         = "low_resolution"
	SignalNoCameraMetadata      = "no_camera_metadata"
	SignalContentCredentials    = "content_credentials_present"
)

// Verdict is the result of screening one file.
type Verdict struct {
	Rejected bool
	Reasons  []string
	Signals  []string
}

// Status is "flagged" when the reviewer has signals to look at, else "passed".
func (v Verdict) Status() string {
	if len(v.Signals) > 0 {
		return "flagged"
	}
	return "passed"
}

// Generators that write their name into the files they produce.
var aiGenerators = []string{
	"dall-e", "dall·e", "dalle", "openai", "chatgpt", "gpt-4o", "sora", "midjourney", "stable diffusion",
	"stablediffusion", "stability ai", "sdxl", "automatic1111", "comfyui", "invokeai", "novelai", "adobe firefly",
	"firefly", "imagen", "gemini", "bard", "bing image creator", "copilot designer", "image creator from designer",
	"leonardo.ai", "ideogram", "flux.1", "black forest labs", "runway", "grok", "aurora", "playground ai",
	"nightcafe", "craiyon", "dreamstudio", "lexica", "canva magic", "magic media", "krea",
}

// IPTC digital source types that declare generated or composited media.
var syntheticSourceTypes = []string{"trainedalgorithmicmedia", "compositesynthetic", "algorithmicmedia", "compositewithtrainedalgorithmicmedia"}

// Editors whose presence in the metadata is worth a reviewer's look.
var editors = []string{
	"photoshop", "gimp", "canva", "picsart", "snapseed", "lightroom", "affinity", "pixlr", "photopea", "paint.net",
	"ilovepdf", "smallpdf", "sejda", "pdfescape", "pdf-xchange", "foxit phantompdf", "nitro pro", "pdffiller", "docfly",
}

var (
	pdfScript     = regexp.MustCompile(`/(JavaScript|JS|Launch|EmbeddedFile|EmbeddedFiles|RichMedia|XFA)[\s(<\[/>]`)
	pdfCreation   = regexp.MustCompile(`/CreationDate\s*\(D:(\d{8,14})`)
	pdfModified   = regexp.MustCompile(`/ModDate\s*\(D:(\d{8,14})`)
	pdfInfoFields = regexp.MustCompile(`/(Producer|Creator|Author|Title|Subject|Keywords)\s*\(([^)]{0,300})\)`)
	xmpTools      = regexp.MustCompile(`(?i)<(?:xmp:CreatorTool|pdf:Producer|exif:Software|tiff:Software|photoshop:Credit|dc:creator)[^>]*>([\s\S]{0,300}?)</`)
)

// Screen checks one file whose sniffed content type is application/pdf,
// image/jpeg or image/png.
func Screen(contentType string, data []byte) Verdict {
	var verdict Verdict
	lower := bytes.ToLower(data)

	// Content credentials and IPTC source types, in any of the formats.
	hasManifest := bytes.Contains(lower, []byte("c2pa")) && (bytes.Contains(lower, []byte("jumb")) || bytes.Contains(lower, []byte("cabx")) || bytes.Contains(lower, []byte("c2pa.claim")))
	if containsAny(lower, syntheticSourceTypes) {
		verdict.reject(ReasonAIProvenance)
	} else if hasManifest {
		verdict.signal(SignalContentCredentials)
		// A manifest names the tool that made the file (searched only around
		// the manifest, so compressed image data cannot match by chance).
		if manifestNamesGenerator(lower) {
			verdict.reject(ReasonAIProvenance)
		}
	}

	switch contentType {
	case "application/pdf":
		screenPDF(data, lower, &verdict)
	case "image/png":
		screenPNG(data, &verdict)
		screenImage(data, lower, &verdict)
	case "image/jpeg":
		screenImage(data, lower, &verdict)
	}
	return verdict
}

func screenPDF(data, lower []byte, verdict *Verdict) {
	// Keys and the document information dictionary live in the PDF's object
	// structure; stream bodies (often compressed) are left out so random
	// bytes can never look like a key.
	structure := withoutStreams(data)
	if pdfScript.Match(structure) {
		verdict.reject(ReasonUnsafePDF)
	}
	tools := ""
	for _, match := range pdfInfoFields.FindAllSubmatch(structure, -1) {
		tools += " " + strings.ToLower(string(match[2]))
	}
	for _, match := range xmpTools.FindAllSubmatch(data, -1) {
		tools += " " + strings.ToLower(string(match[1]))
	}
	if containsAnyString(tools, aiGenerators) {
		verdict.reject(ReasonAIGenerator)
	}
	if containsAnyString(tools, editors) {
		verdict.signal(SignalEditedWithEditor)
	}
	created := pdfCreation.FindSubmatch(structure)
	modified := pdfModified.FindSubmatch(structure)
	if created != nil && modified != nil && string(created[1][:8]) != string(modified[1][:8]) {
		verdict.signal(SignalModifiedAfterCreation)
	}
	if bytes.Count(lower, []byte("%%eof")) > 1 {
		verdict.signal(SignalIncrementalEdits)
	}
}

// screenPNG reads the PNG text chunks, where generators write their settings
// and prompts (A1111 "parameters", ComfyUI "prompt"/"workflow", InvokeAI).
func screenPNG(data []byte, verdict *Verdict) {
	const signature = "\x89PNG\r\n\x1a\n"
	if len(data) < len(signature) || string(data[:len(signature)]) != signature {
		return
	}
	position := len(signature)
	for position+8 <= len(data) {
		length := int(uint32(data[position])<<24 | uint32(data[position+1])<<16 | uint32(data[position+2])<<8 | uint32(data[position+3]))
		kind := string(data[position+4 : position+8])
		start := position + 8
		end := start + length
		if length < 0 || end+4 > len(data) {
			return
		}
		if kind == "tEXt" || kind == "iTXt" || kind == "zTXt" {
			chunk := strings.ToLower(string(data[start:end]))
			key := chunk
			if index := strings.IndexByte(chunk, 0); index >= 0 {
				key = chunk[:index]
			}
			switch key {
			case "parameters", "prompt", "workflow", "invokeai_metadata", "sd-metadata", "dream", "generation_data":
				verdict.reject(ReasonAIGenerator)
			}
			if containsAnyString(chunk, aiGenerators) {
				verdict.reject(ReasonAIGenerator)
			}
		}
		if kind == "IEND" {
			return
		}
		position = end + 4
	}
}

func screenImage(data, lower []byte, verdict *Verdict) {
	// EXIF and XMP software fields naming a generator.
	tools := ""
	for _, match := range xmpTools.FindAllSubmatch(data, -1) {
		tools += " " + strings.ToLower(string(match[1]))
	}
	if containsAnyString(tools, aiGenerators) || exifSoftwareNames(lower, aiGenerators) {
		verdict.reject(ReasonAIGenerator)
	}
	if containsAnyString(tools, editors) || exifSoftwareNames(lower, editors) {
		verdict.signal(SignalEditedWithEditor)
	}
	if config, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		shorter := config.Width
		if config.Height < shorter {
			shorter = config.Height
		}
		if shorter < 600 {
			verdict.signal(SignalLowResolution)
		}
	}
	// A photo of a document taken with a camera or phone normally names it.
	if !bytes.Contains(lower, []byte("exif")) {
		verdict.signal(SignalNoCameraMetadata)
	}
}

// withoutStreams returns a PDF with the bodies of its streams removed.
func withoutStreams(data []byte) []byte {
	var out bytes.Buffer
	rest := data
	for {
		start := bytes.Index(rest, []byte("stream"))
		if start < 0 {
			out.Write(rest)
			return out.Bytes()
		}
		// "endstream" also contains "stream": keep scanning after it.
		if start >= 3 && string(rest[start-3:start]) == "end" {
			out.Write(rest[:start+6])
			rest = rest[start+6:]
			continue
		}
		out.Write(rest[:start+6])
		end := bytes.Index(rest[start+6:], []byte("endstream"))
		if end < 0 {
			return out.Bytes()
		}
		rest = rest[start+6+end:]
	}
}

// manifestNamesGenerator looks for a generator's name near each content
// credentials (C2PA) label.
func manifestNamesGenerator(lower []byte) bool {
	offset := 0
	for {
		index := bytes.Index(lower[offset:], []byte("c2pa"))
		if index < 0 {
			return false
		}
		at := offset + index
		from, to := at-16*1024, at+64*1024
		if from < 0 {
			from = 0
		}
		if to > len(lower) {
			to = len(lower)
		}
		if containsAny(lower[from:to], aiGenerators) || containsAny(lower[from:to], syntheticSourceTypes) {
			return true
		}
		offset = at + 4
	}
}

// exifSoftwareNames looks for the names right after an EXIF header, where the
// Software and Artist tags are stored as plain text.
func exifSoftwareNames(lower []byte, names []string) bool {
	index := bytes.Index(lower, []byte("exif\x00\x00"))
	if index < 0 {
		return false
	}
	end := index + 64*1024
	if end > len(lower) {
		end = len(lower)
	}
	return containsAny(lower[index:end], names)
}

func containsAny(haystack []byte, needles []string) bool {
	for _, needle := range needles {
		if bytes.Contains(haystack, []byte(needle)) {
			return true
		}
	}
	return false
}

func containsAnyString(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}

func (v *Verdict) reject(reason string) {
	v.Rejected = true
	for _, existing := range v.Reasons {
		if existing == reason {
			return
		}
	}
	v.Reasons = append(v.Reasons, reason)
}

func (v *Verdict) signal(signal string) {
	for _, existing := range v.Signals {
		if existing == signal {
			return
		}
	}
	v.Signals = append(v.Signals, signal)
}
