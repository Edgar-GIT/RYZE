// Package trainer_evidence owns the private storage of trainer-application
// evidence documents. Files never live under a web root and are never named by
// the client: the store generates an opaque key, validates the content by its
// magic bytes (never by the client-provided content type or filename) and
// enforces the configured size bound. A stored file is only reachable through
// the authenticated download endpoints.
package trainer_evidence

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	// ErrUnsupportedContentType indicates the sniffed content is not one of the
	// allow-listed document types. The client-provided content type is never
	// trusted, so a renamed executable is rejected here.
	ErrUnsupportedContentType = errors.New("unsupported evidence content type")
	// ErrTooLarge indicates the content exceeds the configured size bound.
	ErrTooLarge = errors.New("evidence document exceeds the maximum size")
	// ErrEmptyContent indicates the content is empty.
	ErrEmptyContent = errors.New("evidence document is empty")
	// ErrInvalidKey indicates a storage key is malformed or attempts to escape
	// the store root.
	ErrInvalidKey = errors.New("invalid evidence storage key")
	// ErrDocumentNotFound indicates the stored file does not exist.
	ErrDocumentNotFound = errors.New("evidence document not found")
)

// storageKeyPattern is the exact shape of a storage key: the application
// segment, a slash and the store-generated file name. Anything else is
// rejected before touching the filesystem, so a malformed key can never
// traverse out of the store root even if the metadata row were corrupted.
var storageKeyPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(pdf|jpg|png)$`)

// prefixPattern is the exact shape of the application segment.
var prefixPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

// allowedContentTypes maps the sniffed media type to the canonical stored
// extension. Only these types can ever be stored.
var allowedContentTypes = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
}

// sniffBytes is how many leading bytes are inspected for the content type.
const sniffBytes = 512

// SavedFile is the validated metadata of a stored document.
type SavedFile struct {
	ByteSize    int64
	SHA256      string
	ContentType string
}

// Store is the private document storage surface. Save generates the opaque
// storage key; Open resolves one for streaming.
type Store interface {
	Save(ctx context.Context, prefix string, content io.Reader) (string, SavedFile, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}

// FileStore is the filesystem-backed private store. The root directory must
// not be under a web root; files are created with owner-only permissions.
type FileStore struct {
	rootDir  string
	maxBytes int64
}

// NewFileStore builds a filesystem store rooted at rootDir. The configuration
// layer validates both values before startup.
func NewFileStore(rootDir string, maxBytes int64) *FileStore {
	return &FileStore{rootDir: filepath.Clean(rootDir), maxBytes: maxBytes}
}

// Save streams content into the store under a single application segment and
// returns the opaque storage key plus the validated metadata. The content is
// written to a temporary file first and atomically renamed, so a failed or
// interrupted upload can never leave a partial document at a valid key.
func (s *FileStore) Save(ctx context.Context, prefix string, content io.Reader) (string, SavedFile, error) {
	if err := ctx.Err(); err != nil {
		return "", SavedFile{}, err
	}
	if !prefixPattern.MatchString(prefix) {
		return "", SavedFile{}, ErrInvalidKey
	}

	directory := filepath.Join(s.rootDir, prefix)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", SavedFile{}, fmt.Errorf("failed to prepare evidence storage: %w", err)
	}

	temporary, err := os.CreateTemp(directory, ".pending-*")
	if err != nil {
		return "", SavedFile{}, fmt.Errorf("failed to open evidence storage: %w", err)
	}
	temporaryPath := temporary.Name()
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(temporaryPath)
		}
	}()
	defer temporary.Close()

	buffered := bufio.NewReader(content)
	header := make([]byte, sniffBytes)
	headerLength, err := io.ReadFull(buffered, header)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", SavedFile{}, fmt.Errorf("failed to read evidence document: %w", err)
	}
	if headerLength == 0 {
		return "", SavedFile{}, ErrEmptyContent
	}

	contentType := http.DetectContentType(header[:headerLength])
	extension, allowed := allowedContentTypes[contentType]
	if !allowed {
		return "", SavedFile{}, ErrUnsupportedContentType
	}

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(temporary, hasher), io.LimitReader(buffered, s.maxBytes+1))
	if err != nil {
		return "", SavedFile{}, fmt.Errorf("failed to store evidence document: %w", err)
	}
	if written > s.maxBytes {
		return "", SavedFile{}, ErrTooLarge
	}
	if err := temporary.Sync(); err != nil {
		return "", SavedFile{}, fmt.Errorf("failed to store evidence document: %w", err)
	}

	fileName := uuid.NewString() + extension
	if err := os.Rename(temporaryPath, filepath.Join(directory, fileName)); err != nil {
		return "", SavedFile{}, fmt.Errorf("failed to finalize evidence document: %w", err)
	}
	keep = true

	return prefix + "/" + fileName, SavedFile{
		ByteSize:    written,
		SHA256:      hex.EncodeToString(hasher.Sum(nil)),
		ContentType: contentType,
	}, nil
}

// Open resolves one storage key into a read stream. Keys that are not exactly
// the shape the store generates are rejected before any filesystem access.
func (s *FileStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !storageKeyPattern.MatchString(key) {
		return nil, ErrInvalidKey
	}

	file, err := os.Open(filepath.Join(s.rootDir, filepath.FromSlash(strings.TrimSpace(key))))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to open evidence document: %w", err)
	}
	return file, nil
}
