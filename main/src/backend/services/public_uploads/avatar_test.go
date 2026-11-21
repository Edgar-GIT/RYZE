package public_uploads

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestSaveAvatarCropsScalesAndStripsMetadata(t *testing.T) {
	store := NewStore(t.TempDir(), 5<<20)
	source := image.NewRGBA(image.Rect(0, 0, 900, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 900; x++ {
			source.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 200, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	// Bytes after the image data (where metadata or a payload could hide).
	encoded.WriteString("hidden-trailer")

	key, err := store.SaveAvatar(context.Background(), &encoded)
	if err != nil {
		t.Fatalf("SaveAvatar: %v", err)
	}
	if !AvatarKeyPattern.MatchString(key) || !store.AvatarExists(key) {
		t.Fatalf("unexpected key %q", key)
	}
	stored, err := os.ReadFile(store.Path(key))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), "hidden-trailer") {
		t.Fatal("the stored photo must be re-encoded")
	}
	decoded, err := jpeg.Decode(bytes.NewReader(stored))
	if err != nil || decoded.Bounds().Dx() != AvatarSize || decoded.Bounds().Dy() != AvatarSize {
		t.Fatalf("expected a %dpx square JPEG (%v)", AvatarSize, err)
	}
	if err := store.DeleteAvatar(key); err != nil || store.AvatarExists(key) {
		t.Fatalf("delete: %v", err)
	}
}

func TestSaveAvatarRejectsNonImages(t *testing.T) {
	store := NewStore(t.TempDir(), 5<<20)
	if _, err := store.SaveAvatar(context.Background(), strings.NewReader("%PDF-1.7 not an image")); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}
	if _, err := store.SaveAvatar(context.Background(), bytes.NewReader([]byte{0xff, 0xd8, 0xff, 0xe0, 0, 0})); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("expected ErrInvalidImage, got %v", err)
	}
}
