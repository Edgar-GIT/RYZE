package public_uploads

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/google/uuid"
)

// Profile photos (owner request, 2026-11-20). An uploaded photo is decoded,
// cropped to a centred square, scaled down to AvatarSize and re-encoded as a
// JPEG: the stored file never carries the original's metadata (location,
// camera, author) or anything hidden after the image data.
const (
	AvatarSize = 512
	// MaxAvatarDimension bounds the decoded picture, so a small file cannot
	// expand into an enormous bitmap.
	MaxAvatarDimension = 6000
)

// AvatarKeyPattern is the only shape an avatar key can have.
var AvatarKeyPattern = regexp.MustCompile(`^avatars/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.jpg$`)

// SaveAvatar stores one profile photo (JPEG or PNG) and returns its key
// ("avatars/<uuid>.jpg").
func (s *Store) SaveAvatar(ctx context.Context, content io.Reader) (string, error) {
	source, err := s.decodeUpload(ctx, content)
	if err != nil {
		return "", err
	}
	return s.storeJPEG("avatars", squareThumbnail(source, AvatarSize))
}

// decodeUpload reads a bounded JPEG or PNG upload and decodes it.
func (s *Store) decodeUpload(ctx context.Context, content io.Reader) (image.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(bufio.NewReader(content), s.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}
	if len(data) == 0 {
		return nil, ErrEmpty
	}
	if int64(len(data)) > s.maxBytes {
		return nil, ErrTooLarge
	}
	switch http.DetectContentType(data) {
	case "image/jpeg", "image/png":
	default:
		return nil, ErrUnsupportedType
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > MaxAvatarDimension || config.Height > MaxAvatarDimension {
		return nil, ErrInvalidImage
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidImage
	}
	return source, nil
}

// storeJPEG re-encodes a picture as a JPEG in folder (no original metadata)
// and returns its key ("<folder>/<uuid>.jpg").
func (s *Store) storeJPEG(folder string, picture image.Image) (string, error) {
	directory := filepath.Join(s.root, folder)
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return "", fmt.Errorf("failed to prepare uploads: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".pending-*")
	if err != nil {
		return "", fmt.Errorf("failed to open uploads: %w", err)
	}
	temporaryPath := temporary.Name()
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(temporaryPath)
		}
	}()
	defer temporary.Close()
	if err := jpeg.Encode(temporary, picture, &jpeg.Options{Quality: 86}); err != nil {
		return "", fmt.Errorf("failed to store image: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return "", fmt.Errorf("failed to store image: %w", err)
	}
	if err := temporary.Chmod(0o640); err != nil {
		return "", fmt.Errorf("failed to store image: %w", err)
	}
	name := uuid.NewString() + ".jpg"
	if err := os.Rename(temporaryPath, filepath.Join(directory, name)); err != nil {
		return "", fmt.Errorf("failed to finalize image: %w", err)
	}
	keep = true
	return folder + "/" + name, nil
}

// DeleteAvatar removes a stored photo. An unknown or malformed key is ignored.
func (s *Store) DeleteAvatar(key string) error {
	if !AvatarKeyPattern.MatchString(key) {
		return nil
	}
	if err := os.Remove(s.Path(key)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// AvatarExists reports whether a served avatar key names an existing file.
func (s *Store) AvatarExists(key string) bool {
	if !AvatarKeyPattern.MatchString(key) {
		return false
	}
	info, err := os.Stat(s.Path(key))
	return err == nil && info.Mode().IsRegular()
}

// squareThumbnail crops the centred square of src and scales it to size ×
// size by averaging each destination pixel's source area (box filter).
func squareThumbnail(src image.Image, size int) *image.RGBA {
	bounds := src.Bounds()
	side := bounds.Dx()
	if bounds.Dy() < side {
		side = bounds.Dy()
	}
	x0 := bounds.Min.X + (bounds.Dx()-side)/2
	y0 := bounds.Min.Y + (bounds.Dy()-side)/2
	out := size
	if side < size {
		out = side
	}
	dst := image.NewRGBA(image.Rect(0, 0, out, out))
	for y := 0; y < out; y++ {
		sy0 := y0 + y*side/out
		sy1 := y0 + (y+1)*side/out
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < out; x++ {
			sx0 := x0 + x*side/out
			sx1 := x0 + (x+1)*side/out
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var r, g, b, n uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					cr, cg, cb, ca := src.At(sx, sy).RGBA()
					// Transparent areas become white, not black.
					r += uint64(cr) + uint64(0xffff-ca)
					g += uint64(cg) + uint64(0xffff-ca)
					b += uint64(cb) + uint64(0xffff-ca)
					n++
				}
			}
			dst.SetRGBA(x, y, color.RGBA{R: uint8(r / n >> 8), G: uint8(g / n >> 8), B: uint8(b / n >> 8), A: 0xff})
		}
	}
	return dst
}
