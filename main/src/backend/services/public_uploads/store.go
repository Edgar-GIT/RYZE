// Package public_uploads owns the public uploads directory: programme cover images
// today (uploads/programs/{generic,l1,l2}), with room for profile photos
// (uploads/avatars) and feedback photos (uploads/feedback) later. Files are
// public by design (they illustrate the catalogue), but they are never named or
// typed by the client: the store generates an opaque UUID name, accepts only
// JPEG, PNG and WebP by their magic bytes, checks the image header, enforces the
// size bound and writes atomically.
package public_uploads

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // registers the JPEG header decoder
	_ "image/png"  // registers the PNG header decoder
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	// ErrUnsupportedType indicates the content is not a JPEG, PNG or WebP image.
	ErrUnsupportedType = errors.New("unsupported image type")
	// ErrTooLarge indicates the image exceeds the configured size bound.
	ErrTooLarge = errors.New("image exceeds the maximum size")
	// ErrEmpty indicates the upload is empty.
	ErrEmpty = errors.New("image is empty")
	// ErrInvalidImage indicates the bytes claim an image type but its header
	// cannot be decoded or its dimensions are out of bounds.
	ErrInvalidImage = errors.New("invalid image")
	// ErrInvalidFolder indicates an unknown destination folder.
	ErrInvalidFolder = errors.New("invalid uploads folder")
)

// MaxImageDimension bounds both sides of an accepted image.
const MaxImageDimension = 8000

// ProgramFolders are the programme cover folders, one per catalogue category.
var ProgramFolders = []string{"generic", "l1", "l2"}

// ReservedFolders are created at startup so the uploads layout is visible on
// disk; avatars and feedback are reserved for the planned profile and feedback
// photos and are not written by any endpoint yet.
var ReservedFolders = []string{"avatars", "feedback"}

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// ServedKeyPattern is the exact shape of a public programme image key.
var ServedKeyPattern = regexp.MustCompile(`^programs/(generic|l1|l2)/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(jpg|png|webp)$`)

const sniffBytes = 512

// Store is the filesystem-backed public uploads store.
type Store struct {
	root     string
	maxBytes int64
}

// NewStore builds a store rooted at dir.
func NewStore(dir string, maxBytes int64) *Store {
	return &Store{root: filepath.Clean(dir), maxBytes: maxBytes}
}

// Root returns the store root directory.
func (s *Store) Root() string { return s.root }

// MaxBytes returns the image size bound.
func (s *Store) MaxBytes() int64 { return s.maxBytes }

// EnsureLayout creates the programme folders and the reserved folders.
func (s *Store) EnsureLayout() error {
	for _, folder := range ProgramFolders {
		if err := os.MkdirAll(filepath.Join(s.root, "programs", folder), 0o750); err != nil {
			return fmt.Errorf("failed to prepare uploads: %w", err)
		}
	}
	for _, folder := range ReservedFolders {
		if err := os.MkdirAll(filepath.Join(s.root, folder), 0o750); err != nil {
			return fmt.Errorf("failed to prepare uploads: %w", err)
		}
	}
	return nil
}

// SaveProgramCover stores one cover image in programs/<folder>/ and returns its
// key ("programs/<folder>/<uuid>.<ext>").
func (s *Store) SaveProgramCover(ctx context.Context, folder string, content io.Reader) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	valid := false
	for _, candidate := range ProgramFolders {
		if folder == candidate {
			valid = true
			break
		}
	}
	if !valid {
		return "", ErrInvalidFolder
	}

	directory := filepath.Join(s.root, "programs", folder)
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return "", fmt.Errorf("failed to prepare uploads: %w", err)
	}

	buffered := bufio.NewReader(content)
	header := make([]byte, sniffBytes)
	headerLength, err := io.ReadFull(buffered, header)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("failed to read image: %w", err)
	}
	if headerLength == 0 {
		return "", ErrEmpty
	}
	contentType := http.DetectContentType(header[:headerLength])
	extension, allowed := allowedImageTypes[contentType]
	if !allowed {
		return "", ErrUnsupportedType
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

	if _, err := temporary.Write(header[:headerLength]); err != nil {
		return "", fmt.Errorf("failed to store image: %w", err)
	}
	written := int64(headerLength)
	remaining, err := io.Copy(temporary, io.LimitReader(buffered, s.maxBytes+1-int64(headerLength)))
	if err != nil {
		return "", fmt.Errorf("failed to store image: %w", err)
	}
	written += remaining
	if written > s.maxBytes {
		return "", ErrTooLarge
	}

	// JPEG and PNG headers are decoded to prove the file is a real image with
	// sane dimensions; WebP is accepted on its RIFF/WEBP signature because the
	// standard library carries no WebP decoder.
	if extension != ".webp" {
		if _, err := temporary.Seek(0, io.SeekStart); err != nil {
			return "", fmt.Errorf("failed to verify image: %w", err)
		}
		config, _, err := image.DecodeConfig(temporary)
		if err != nil || config.Width <= 0 || config.Height <= 0 ||
			config.Width > MaxImageDimension || config.Height > MaxImageDimension {
			return "", ErrInvalidImage
		}
	} else if !bytes.Equal(header[8:12], []byte("WEBP")) {
		return "", ErrInvalidImage
	}

	if err := temporary.Sync(); err != nil {
		return "", fmt.Errorf("failed to store image: %w", err)
	}
	if err := temporary.Chmod(0o640); err != nil {
		return "", fmt.Errorf("failed to store image: %w", err)
	}
	fileName := uuid.NewString() + extension
	if err := os.Rename(temporaryPath, filepath.Join(directory, fileName)); err != nil {
		return "", fmt.Errorf("failed to finalize image: %w", err)
	}
	keep = true
	return "programs/" + folder + "/" + fileName, nil
}

// Exists reports whether a served key names an existing file.
func (s *Store) Exists(key string) bool {
	if !ServedKeyPattern.MatchString(key) {
		return false
	}
	info, err := os.Stat(s.Path(key))
	return err == nil && info.Mode().IsRegular()
}

// Path resolves a validated key to its file path. Callers must validate the
// key with ServedKeyPattern first.
func (s *Store) Path(key string) string {
	return filepath.Join(s.root, filepath.FromSlash(strings.TrimSpace(key)))
}

// ContentTypeFor returns the content type of a served key's extension.
func ContentTypeFor(key string) string {
	switch filepath.Ext(key) {
	case ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
