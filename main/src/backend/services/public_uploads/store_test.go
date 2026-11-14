package public_uploads_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	uploads "ryze/backend/services/public_uploads"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{0, 203, 255, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestSaveProgramCover(t *testing.T) {
	root := t.TempDir()
	store := uploads.NewStore(root, 1<<20)
	if err := store.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout: %v", err)
	}
	for _, folder := range []string{"programs/generic", "programs/l1", "programs/l2", "avatars", "feedback"} {
		if info, err := os.Stat(filepath.Join(root, folder)); err != nil || !info.IsDir() {
			t.Fatalf("folder %s missing", folder)
		}
	}

	key, err := store.SaveProgramCover(context.Background(), "l1", bytes.NewReader(pngBytes(t, 4, 3)))
	if err != nil {
		t.Fatalf("SaveProgramCover: %v", err)
	}
	if !strings.HasPrefix(key, "programs/l1/") || !strings.HasSuffix(key, ".png") || !uploads.ServedKeyPattern.MatchString(key) {
		t.Fatalf("unexpected key %q", key)
	}
	if !store.Exists(key) {
		t.Fatalf("stored file not found for %q", key)
	}
	if store.Exists("programs/l1/../../etc/passwd") {
		t.Fatal("a traversal key must never resolve")
	}
}

func TestSaveProgramCoverRejects(t *testing.T) {
	store := uploads.NewStore(t.TempDir(), 2048)
	cases := []struct {
		name    string
		folder  string
		content []byte
		want    error
	}{
		{"script disguised as image", "generic", []byte("<script>alert(1)</script>"), uploads.ErrUnsupportedType},
		{"empty", "generic", nil, uploads.ErrEmpty},
		{"unknown folder", "../etc", pngBytes(t, 2, 2), uploads.ErrInvalidFolder},
		{"png signature without a valid header", "generic", append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...), uploads.ErrInvalidImage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := store.SaveProgramCover(context.Background(), tc.folder, bytes.NewReader(tc.content))
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestSaveProgramCoverSizeBound(t *testing.T) {
	store := uploads.NewStore(t.TempDir(), 40)
	if _, err := store.SaveProgramCover(context.Background(), "generic", bytes.NewReader(pngBytes(t, 8, 8))); !errors.Is(err, uploads.ErrTooLarge) {
		t.Fatalf("got %v, want ErrTooLarge", err)
	}
}
