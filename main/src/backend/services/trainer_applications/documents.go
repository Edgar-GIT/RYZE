package trainer_applications

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/trainer_evidence"
)

var (
	// ErrDocumentNotFound indicates the requested evidence document does not
	// exist, is soft-deleted, or belongs to another applicant. The three cases
	// are deliberately indistinguishable so the response can never confirm the
	// existence of another user's health-adjacent evidence.
	ErrDocumentNotFound = errors.New("trainer application document not found")
	// ErrDocumentUnsupportedType indicates the uploaded content is not one of
	// the allow-listed document types (validated by magic bytes, never by the
	// client-provided content type or filename).
	ErrDocumentUnsupportedType = errors.New("trainer application document type is not allowed")
	// ErrDocumentTooLarge indicates the uploaded content exceeds the configured
	// per-document size bound.
	ErrDocumentTooLarge = errors.New("trainer application document exceeds the maximum size")
)

// DocumentRepository is the data-access surface required for trainer-application
// evidence metadata. The binary content is owned by trainer_evidence.Store.
type DocumentRepository interface {
	Create(ctx context.Context, document *models.TrainerApplicationDocument) error
	ListByApplication(ctx context.Context, applicationID string) ([]models.TrainerApplicationDocument, error)
	FindByID(ctx context.Context, documentID string) (*models.TrainerApplicationDocument, error)
	SoftDelete(ctx context.Context, documentID string) error
}

// DecisionRepository is the read-only data-access surface for the append-only
// review decision history. Decisions are written by the application
// repository inside the review transaction, never directly by the service.
type DecisionRepository interface {
	ListByApplication(ctx context.Context, applicationID string) ([]models.TrainerApplicationDecision, error)
}

const (
	maxOriginalFilenameLength = 255
	maxAdminIDLength          = 40
	maxDecisionReasonLength   = 500
)

// UploadDocument stores one validated evidence document for the caller's
// PENDING application. The application is always resolved from the
// authenticated user id, never from client input, and evidence can only be
// attached while the application is still PENDING: once reviewed, the
// submission is locked.
func (s *service) UploadDocument(ctx context.Context, userID, kind, originalFilename string, content io.Reader) (*models.TrainerApplicationDocument, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validateDocumentKind(kind); err != nil {
		return nil, err
	}

	application, err := s.applications.FindActiveByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrApplicationNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, fmt.Errorf("failed to load active trainer application: %w", err)
	}
	if application.Status != models.ApplicationStatusPending {
		return nil, ErrApplicationStateConflict
	}

	storageKey, saved, err := s.evidence.Save(ctx, application.ID, content)
	if err != nil {
		switch {
		case errors.Is(err, trainer_evidence.ErrUnsupportedContentType):
			return nil, ErrDocumentUnsupportedType
		case errors.Is(err, trainer_evidence.ErrTooLarge):
			return nil, ErrDocumentTooLarge
		case errors.Is(err, trainer_evidence.ErrEmptyContent):
			return nil, fmt.Errorf("%w: document is empty", ErrInvalidInput)
		default:
			return nil, fmt.Errorf("failed to store trainer application document: %w", err)
		}
	}

	document := &models.TrainerApplicationDocument{
		ApplicationID:    application.ID,
		Kind:             strings.TrimSpace(kind),
		OriginalFilename: sanitizeOriginalFilename(originalFilename),
		StorageKey:       storageKey,
		ContentType:      saved.ContentType,
		ByteSize:         saved.ByteSize,
		SHA256:           saved.SHA256,
	}
	if err := s.documents.Create(ctx, document); err != nil {
		// The file is kept: the metadata row can be reconstructed from it and a
		// failed upload must never destroy the only copy of the evidence.
		log.Printf("trainer application %s: stored evidence file %s but failed to record metadata: %v", application.ID, storageKey, err)
		return nil, fmt.Errorf("failed to record trainer application document: %w", err)
	}
	return document, nil
}

// ListOwnDocuments returns the caller's evidence documents for their latest
// application, including a rejected one, so the applicant can see what is on
// file. The user id always comes from the authenticated session.
func (s *service) ListOwnDocuments(ctx context.Context, userID string) ([]models.TrainerApplicationDocument, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	application, err := s.applications.FindLatestByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrApplicationNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, fmt.Errorf("failed to load trainer application: %w", err)
	}
	documents, err := s.documents.ListByApplication(ctx, application.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list trainer application documents: %w", err)
	}
	return documents, nil
}

// GetOwnDocument returns one active document owned by the caller.
func (s *service) GetOwnDocument(ctx context.Context, userID, documentID string) (*models.TrainerApplicationDocument, error) {
	document, _, err := s.findOwnedDocument(ctx, userID, documentID)
	return document, err
}

// OpenOwnDocument resolves one active document owned by the caller into a read
// stream for the authenticated download endpoint.
func (s *service) OpenOwnDocument(ctx context.Context, userID, documentID string) (*models.TrainerApplicationDocument, io.ReadCloser, error) {
	document, _, err := s.findOwnedDocument(ctx, userID, documentID)
	if err != nil {
		return nil, nil, err
	}
	content, err := s.openStoredDocument(ctx, document)
	if err != nil {
		return nil, nil, err
	}
	return document, content, nil
}

// DeleteOwnDocument withdraws one document while the application is PENDING.
// Only the metadata row is soft-deleted; the file is kept so the review trail
// can never be destroyed. Documents of a reviewed application are locked and
// can no longer be withdrawn.
func (s *service) DeleteOwnDocument(ctx context.Context, userID, documentID string) error {
	document, application, err := s.findOwnedDocument(ctx, userID, documentID)
	if err != nil {
		return err
	}
	if application.Status != models.ApplicationStatusPending {
		return ErrApplicationStateConflict
	}
	if err := s.documents.SoftDelete(ctx, document.ID); err != nil {
		if errors.Is(err, repositories.ErrApplicationDocumentNotFound) || errors.Is(err, repositories.ErrApplicationDocumentAlreadyRemoved) {
			return ErrDocumentNotFound
		}
		return fmt.Errorf("failed to remove trainer application document: %w", err)
	}
	return nil
}

// ListApplicationDocuments returns the active evidence documents of one
// application for a reviewer. The application must exist.
func (s *service) ListApplicationDocuments(ctx context.Context, applicationID string) ([]models.TrainerApplicationDocument, error) {
	if err := validateApplicationID(applicationID); err != nil {
		return nil, err
	}
	if _, err := s.applications.FindByID(ctx, applicationID); err != nil {
		if errors.Is(err, repositories.ErrApplicationNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, fmt.Errorf("failed to load trainer application: %w", err)
	}
	documents, err := s.documents.ListByApplication(ctx, applicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list trainer application documents: %w", err)
	}
	return documents, nil
}

// GetApplicationDocument returns one document for a reviewer, scoped to the
// reviewed application so a document id can never be replayed across
// applications.
func (s *service) GetApplicationDocument(ctx context.Context, applicationID, documentID string) (*models.TrainerApplicationDocument, error) {
	document, err := s.findReviewDocument(ctx, applicationID, documentID)
	return document, err
}

// OpenApplicationDocument resolves one reviewed document into a read stream for
// the authenticated reviewer download endpoint.
func (s *service) OpenApplicationDocument(ctx context.Context, applicationID, documentID string) (*models.TrainerApplicationDocument, io.ReadCloser, error) {
	document, err := s.findReviewDocument(ctx, applicationID, documentID)
	if err != nil {
		return nil, nil, err
	}
	content, err := s.openStoredDocument(ctx, document)
	if err != nil {
		return nil, nil, err
	}
	return document, content, nil
}

// ListDecisions returns the append-only review history of one application.
func (s *service) ListDecisions(ctx context.Context, applicationID string) ([]models.TrainerApplicationDecision, error) {
	if err := validateApplicationID(applicationID); err != nil {
		return nil, err
	}
	if _, err := s.applications.FindByID(ctx, applicationID); err != nil {
		if errors.Is(err, repositories.ErrApplicationNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, fmt.Errorf("failed to load trainer application: %w", err)
	}
	decisions, err := s.decisions.ListByApplication(ctx, applicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list trainer application decisions: %w", err)
	}
	return decisions, nil
}

func (s *service) findOwnedDocument(ctx context.Context, userID, documentID string) (*models.TrainerApplicationDocument, *models.TrainerApplication, error) {
	if err := validateUserID(userID); err != nil {
		return nil, nil, err
	}
	if err := validateDocumentID(documentID); err != nil {
		return nil, nil, err
	}

	document, err := s.documents.FindByID(ctx, documentID)
	if err != nil {
		if errors.Is(err, repositories.ErrApplicationDocumentNotFound) {
			return nil, nil, ErrDocumentNotFound
		}
		return nil, nil, fmt.Errorf("failed to load trainer application document: %w", err)
	}

	application, err := s.applications.FindByID(ctx, document.ApplicationID)
	if err != nil {
		if errors.Is(err, repositories.ErrApplicationNotFound) {
			return nil, nil, ErrDocumentNotFound
		}
		return nil, nil, fmt.Errorf("failed to load trainer application: %w", err)
	}
	if application.UserID != userID {
		return nil, nil, ErrDocumentNotFound
	}
	return document, application, nil
}

func (s *service) findReviewDocument(ctx context.Context, applicationID, documentID string) (*models.TrainerApplicationDocument, error) {
	if err := validateApplicationID(applicationID); err != nil {
		return nil, err
	}
	if err := validateDocumentID(documentID); err != nil {
		return nil, err
	}

	document, err := s.documents.FindByID(ctx, documentID)
	if err != nil {
		if errors.Is(err, repositories.ErrApplicationDocumentNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to load trainer application document: %w", err)
	}
	if document.ApplicationID != applicationID {
		return nil, ErrDocumentNotFound
	}
	return document, nil
}

func (s *service) openStoredDocument(ctx context.Context, document *models.TrainerApplicationDocument) (io.ReadCloser, error) {
	content, err := s.evidence.Open(ctx, document.StorageKey)
	if err != nil {
		if errors.Is(err, trainer_evidence.ErrDocumentNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to open trainer application document: %w", err)
	}
	return content, nil
}

func validateDocumentKind(kind string) error {
	switch strings.TrimSpace(kind) {
	case models.DocumentKindCertification, models.DocumentKindIdentification, models.DocumentKindInsurance, models.DocumentKindOther:
		return nil
	default:
		return fmt.Errorf("%w: invalid document kind", ErrInvalidInput)
	}
}

func validateDocumentID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: document id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid document id", ErrInvalidInput)
	}
	return nil
}

func validateAdminID(adminID string) error {
	trimmed := strings.TrimSpace(adminID)
	if trimmed == "" {
		return fmt.Errorf("%w: reviewer identity is required", ErrInvalidInput)
	}
	if len(trimmed) > maxAdminIDLength {
		return fmt.Errorf("%w: reviewer identity is too long", ErrInvalidInput)
	}
	return nil
}

func validateReason(reason string) error {
	if utf8.RuneCountInString(normalizeReason(reason)) > maxDecisionReasonLength {
		return fmt.Errorf("%w: reason must be at most %d characters", ErrInvalidInput, maxDecisionReasonLength)
	}
	return nil
}

// normalizeReason collapses control characters and surrounding whitespace so a
// decision reason is a single stable, printable value. Reasons are reviewer
// notes, never applicant-supplied.
func normalizeReason(reason string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, reason)
	return strings.TrimSpace(cleaned)
}

// sanitizeOriginalFilename keeps the applicant's filename as harmless display
// metadata only: path components and control characters are stripped and the
// value is bounded because it is stored and rendered in the review UI.
func sanitizeOriginalFilename(name string) string {
	base := filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	cleaned := strings.Map(func(r rune) rune {
		if r == 0 || unicode.IsControl(r) {
			return -1
		}
		return r
	}, base)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return "document"
	}
	runes := []rune(cleaned)
	if len(runes) > maxOriginalFilenameLength {
		cleaned = string(runes[:maxOriginalFilenameLength])
	}
	return cleaned
}
