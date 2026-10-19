package trainer_evidence_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"ryze/backend/services/trainer_evidence"
)

const (
	testPrefix   = "33333333-3333-3333-3333-333333333333"
	testMaxBytes = 1 << 20
)

var keyPattern = regexp.MustCompile(`^` + regexp.QuoteMeta(testPrefix) + `/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(pdf|jpg|png)$`)

func pdfPayload() []byte {
	return append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte{'x'}, 256)...)
}

func jpegPayload() []byte {
	return append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}, bytes.Repeat([]byte{0x00}, 64)...)
}

func pngPayload() []byte {
	return append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte{0x00}, 64)...)
}

func newStore(t *testing.T) (*trainer_evidence.FileStore, string) {
	t.Helper()
	root := t.TempDir()
	return trainer_evidence.NewFileStore(root, testMaxBytes), root
}

func TestFileStoreSaveAndOpen(t *testing.T) {
	store, root := newStore(t)
	ctx := context.Background()

	for _, tc := range []struct {
		name        string
		content     []byte
		extension   string
		contentType string
	}{
		{name: "pdf", content: pdfPayload(), extension: ".pdf", contentType: "application/pdf"},
		{name: "jpeg", content: jpegPayload(), extension: ".jpg", contentType: "image/jpeg"},
		{name: "png", content: pngPayload(), extension: ".png", contentType: "image/png"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key, saved, err := store.Save(ctx, testPrefix, bytes.NewReader(tc.content))
			if err != nil {
				t.Fatalf("Save: %v", err)
			}
			if !keyPattern.MatchString(key) || !strings.HasSuffix(key, tc.extension) {
				t.Fatalf("unexpected storage key %q", key)
			}
			if saved.ContentType != tc.contentType {
				t.Fatalf("expected content type %q, got %q", tc.contentType, saved.ContentType)
			}
			if saved.ByteSize != int64(len(tc.content)) {
				t.Fatalf("expected byte size %d, got %d", len(tc.content), saved.ByteSize)
			}
			sum := sha256.Sum256(tc.content)
			if saved.SHA256 != hex.EncodeToString(sum[:]) {
				t.Fatalf("unexpected sha256 %q", saved.SHA256)
			}

			// The stored file carries the validated content and is private.
			storedPath := filepath.Join(root, filepath.FromSlash(key))
			info, err := os.Stat(storedPath)
			if err != nil {
				t.Fatalf("stat stored file: %v", err)
			}
			if info.Mode().Perm() != 0o600 {
				t.Fatalf("expected owner-only permissions, got %v", info.Mode().Perm())
			}
			storedBytes, err := os.ReadFile(storedPath)
			if err != nil {
				t.Fatalf("read stored file: %v", err)
			}
			if !bytes.Equal(storedBytes, tc.content) {
				t.Fatal("stored bytes differ from the upload")
			}

			content, err := store.Open(ctx, key)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer content.Close()
			streamed, err := io.ReadAll(content)
			if err != nil {
				t.Fatalf("read stream: %v", err)
			}
			if !bytes.Equal(streamed, tc.content) {
				t.Fatal("streamed bytes differ from the stored content")
			}
		})
	}
}

func TestFileStoreRejectsUnsupportedContent(t *testing.T) {
	store, root := newStore(t)

	_, _, err := store.Save(context.Background(), testPrefix, strings.NewReader("#!/bin/sh\nrm -rf /"))
	if !errors.Is(err, trainer_evidence.ErrUnsupportedContentType) {
		t.Fatalf("expected ErrUnsupportedContentType, got %v", err)
	}

	entries := pendingEntries(t, root)
	if len(entries) != 0 {
		t.Fatalf("a rejected upload must leave no file behind, found %v", entries)
	}
}

func TestFileStoreRejectsEmptyAndOversizedContent(t *testing.T) {
	store, root := newStore(t)

	if _, _, err := store.Save(context.Background(), testPrefix, bytes.NewReader(nil)); !errors.Is(err, trainer_evidence.ErrEmptyContent) {
		t.Fatalf("expected ErrEmptyContent, got %v", err)
	}

	oversized := append(pdfPayload(), bytes.Repeat([]byte{'y'}, testMaxBytes)...)
	if _, _, err := store.Save(context.Background(), testPrefix, bytes.NewReader(oversized)); !errors.Is(err, trainer_evidence.ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
	if entries := pendingEntries(t, root); len(entries) != 0 {
		t.Fatalf("a rejected upload must leave no file behind, found %v", entries)
	}

	// Exactly at the bound is accepted.
	atBound := make([]byte, testMaxBytes)
	copy(atBound, pdfPayload())
	if _, saved, err := store.Save(context.Background(), testPrefix, bytes.NewReader(atBound)); err != nil {
		t.Fatalf("at-bound Save: %v", err)
	} else if saved.ByteSize != testMaxBytes {
		t.Fatalf("expected byte size %d, got %d", testMaxBytes, saved.ByteSize)
	}
}

func TestFileStoreRejectsInvalidPrefix(t *testing.T) {
	store, _ := newStore(t)

	for _, prefix := range []string{"", "..", "../evil", "a/b", "a\\b", strings.Repeat("a", 65)} {
		if _, _, err := store.Save(context.Background(), prefix, bytes.NewReader(pdfPayload())); !errors.Is(err, trainer_evidence.ErrInvalidKey) {
			t.Fatalf("prefix %q: expected ErrInvalidKey, got %v", prefix, err)
		}
	}
}

func TestFileStoreOpenRejectsInvalidKeys(t *testing.T) {
	store, _ := newStore(t)

	for _, key := range []string{
		"",
		"../../etc/passwd",
		"abc/../../etc/passwd",
		"33333333-3333-3333-3333-333333333333/../../etc/passwd",
		"33333333-3333-3333-3333-333333333333/not-a-uuid.pdf",
		"33333333-3333-3333-3333-333333333333/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.sh",
		"33333333-3333-3333-3333-333333333333/nested/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.pdf",
	} {
		if _, err := store.Open(context.Background(), key); !errors.Is(err, trainer_evidence.ErrInvalidKey) {
			t.Fatalf("key %q: expected ErrInvalidKey, got %v", key, err)
		}
	}
}

func TestFileStoreOpenMissingFile(t *testing.T) {
	store, _ := newStore(t)

	missing := testPrefix + "/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.pdf"
	if _, err := store.Open(context.Background(), missing); !errors.Is(err, trainer_evidence.ErrDocumentNotFound) {
		t.Fatalf("expected ErrDocumentNotFound, got %v", err)
	}
}

// pendingEntries lists every file and directory inside the store root,
// including the temporary upload files a failed Save must clean up.
func pendingEntries(t *testing.T, root string) []string {
	t.Helper()
	var entries []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if !info.IsDir() {
			entries = append(entries, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk store root: %v", err)
	}
	return entries
}
