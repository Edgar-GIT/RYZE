package trainer_applications

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/document_screening"
)

// The complete application (owner request, 2026-11-20): who the applicant is,
// their background and their evidence, submitted together. Every file is
// screened before anything is stored; one rejected file rejects the whole
// submission, with the reason per file, so nothing half-submitted remains.

const (
	MaxSubmittedDocuments = 6
	maxWorkEntries        = 10
	maxEducationEntries   = 10
	minAbout              = 50
	maxAbout              = 4000
	minimumAge            = 18
)

// ReasonDuplicateDocument rejects a file already submitted by another account.
const ReasonDuplicateDocument = "duplicate_document"

// WorkEntry is one place the applicant worked.
type WorkEntry struct {
	Organization string `json:"organization"`
	Role         string `json:"role"`
	StartYear    int    `json:"start_year"`
	EndYear      *int   `json:"end_year"`
}

// EducationEntry is one school, course or qualification.
type EducationEntry struct {
	Institution   string `json:"institution"`
	Qualification string `json:"qualification"`
	Year          int    `json:"year"`
}

// ApplicationForm is what the applicant writes.
type ApplicationForm struct {
	FullName        string
	BirthDate       string // YYYY-MM-DD
	City            string
	Profession      string
	YearsExperience int
	WorkHistory     []WorkEntry
	Education       []EducationEntry
	About           string
}

// SubmittedDocument is one attached file, read in full for the screening.
type SubmittedDocument struct {
	Kind     string
	Filename string
	Data     []byte
}

// FormError lists the form fields that are missing or invalid.
type FormError struct {
	Fields []string
}

func (e *FormError) Error() string {
	return "invalid application form: " + strings.Join(e.Fields, ", ")
}

// DocumentRejection is one refused file and why.
type DocumentRejection struct {
	Filename string   `json:"filename"`
	Reasons  []string `json:"reasons"`
}

// DocumentsRejectedError lists the refused files; nothing was stored.
type DocumentsRejectedError struct {
	Rejections []DocumentRejection
}

func (e *DocumentsRejectedError) Error() string {
	return fmt.Sprintf("%d trainer application document(s) rejected", len(e.Rejections))
}

// ErrMissingDocuments means the required evidence was not attached (at least
// one certification and one identification document).
var ErrMissingDocuments = errors.New("required trainer application documents missing")

// DocumentHashes tells whether another account already submitted a file.
type DocumentHashes interface {
	HashUsedByAnotherUser(ctx context.Context, sha256, userID string) (bool, error)
}

// UsernameLookup finds an active account by its @username.
type UsernameLookup interface {
	FindByUsername(ctx context.Context, username string) (*models.User, error)
}

// Option configures an optional collaborator.
type Option func(*service)

// WithDocumentHashes enables the duplicate-document check.
func WithDocumentHashes(hashes DocumentHashes) Option {
	return func(s *service) { s.hashes = hashes }
}

// WithUsernames enables PromoteUser.
func WithUsernames(usernames UsernameLookup) Option {
	return func(s *service) { s.usernames = usernames }
}

// SubmitApplication validates the form and the documents, screens every file
// and, when everything passes, creates the PENDING application with its
// details and stores the documents with their screening results.
func (s *service) SubmitApplication(ctx context.Context, userID string, form ApplicationForm, documents []SubmittedDocument) (*models.TrainerApplication, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	details, err := validateForm(form, time.Now())
	if err != nil {
		return nil, err
	}
	if err := validateDocumentSet(documents); err != nil {
		return nil, err
	}

	if _, err := s.users.FindByID(ctx, userID); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to check user: %w", err)
	}
	if _, err := s.trainers.FindByUserID(ctx, userID); err == nil {
		return nil, ErrAlreadyTrainer
	} else if !errors.Is(err, repositories.ErrTrainerNotFound) {
		return nil, fmt.Errorf("failed to check trainer profile: %w", err)
	}
	if _, err := s.applications.FindActiveByUserID(ctx, userID); err == nil {
		return nil, ErrApplicationAlreadyActive
	} else if !errors.Is(err, repositories.ErrApplicationNotFound) {
		return nil, fmt.Errorf("failed to check trainer application: %w", err)
	}

	// Screen every file before anything is written.
	verdicts := make([]document_screening.Verdict, len(documents))
	var rejections []DocumentRejection
	for index, document := range documents {
		contentType := http.DetectContentType(document.Data)
		if contentType != "application/pdf" && contentType != "image/jpeg" && contentType != "image/png" {
			return nil, fmt.Errorf("%w: %s", ErrDocumentUnsupportedType, sanitizeOriginalFilename(document.Filename))
		}
		verdict := document_screening.Screen(contentType, document.Data)
		if s.hashes != nil {
			sum := sha256.Sum256(document.Data)
			used, err := s.hashes.HashUsedByAnotherUser(ctx, hex.EncodeToString(sum[:]), userID)
			if err != nil {
				return nil, fmt.Errorf("failed to check document history: %w", err)
			}
			if used {
				verdict.Rejected = true
				verdict.Reasons = append(verdict.Reasons, ReasonDuplicateDocument)
			}
		}
		if verdict.Rejected {
			rejections = append(rejections, DocumentRejection{Filename: sanitizeOriginalFilename(document.Filename), Reasons: verdict.Reasons})
		}
		verdicts[index] = verdict
	}
	if len(rejections) > 0 {
		slog.Warn("trainer application documents rejected by screening", slog.String("user_id", userID), slog.Int("files", len(rejections)))
		return nil, &DocumentsRejectedError{Rejections: rejections}
	}

	application := details
	application.UserID = userID
	application.Status = models.ApplicationStatusPending
	if err := s.applications.Create(ctx, application); err != nil {
		if errors.Is(err, repositories.ErrApplicationAlreadyActive) {
			return nil, ErrApplicationAlreadyActive
		}
		return nil, fmt.Errorf("failed to create trainer application: %w", err)
	}

	for index, document := range documents {
		storageKey, saved, err := s.evidence.Save(ctx, application.ID, bytes.NewReader(document.Data))
		if err == nil {
			signals, _ := json.Marshal(nonNil(verdicts[index].Signals))
			err = s.documents.Create(ctx, &models.TrainerApplicationDocument{
				ApplicationID:    application.ID,
				Kind:             document.Kind,
				OriginalFilename: sanitizeOriginalFilename(document.Filename),
				StorageKey:       storageKey,
				ContentType:      saved.ContentType,
				ByteSize:         saved.ByteSize,
				SHA256:           saved.SHA256,
				ScreeningStatus:  verdicts[index].Status(),
				ScreeningSignals: signals,
			})
		}
		if err != nil {
			// The submission must not stay half-stored: close it so the
			// applicant can submit again.
			if rejectErr := s.applications.Reject(ctx, application.ID, nil); rejectErr != nil {
				slog.Error("incomplete trainer application not closed", slog.String("application_id", application.ID), slog.String("error", rejectErr.Error()))
			}
			return nil, fmt.Errorf("failed to store trainer application document: %w", err)
		}
	}

	created, err := s.applications.FindByID(ctx, application.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload trainer application: %w", err)
	}
	return created, nil
}

// PromoteUser makes the active account with this @username a trainer
// directly (owner request, 2026-11-20): an application is created (or the
// pending one reused) and approved with the administrator's decision recorded,
// through the same transaction that creates the trainer profile.
func (s *service) PromoteUser(ctx context.Context, username, adminID, reason string) (*models.TrainerApplication, error) {
	if s.usernames == nil {
		return nil, ErrUserNotFound
	}
	if err := validateAdminID(adminID); err != nil {
		return nil, err
	}
	name := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(username), "@"))
	if name == "" || len(name) > 30 {
		return nil, ErrUserNotFound
	}
	user, err := s.usernames.FindByUsername(ctx, name)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user.TestPersona != nil {
		return nil, ErrUserNotFound
	}
	if _, err := s.trainers.FindByUserID(ctx, user.ID); err == nil {
		return nil, ErrAlreadyTrainer
	} else if !errors.Is(err, repositories.ErrTrainerNotFound) {
		return nil, fmt.Errorf("failed to check trainer profile: %w", err)
	}

	application, err := s.applications.FindActiveByUserID(ctx, user.ID)
	switch {
	case err == nil && application.Status == models.ApplicationStatusApproved:
		return nil, ErrAlreadyTrainer
	case err == nil:
		// The pending application is the one approved.
	case errors.Is(err, repositories.ErrApplicationNotFound):
		application = &models.TrainerApplication{UserID: user.ID, Status: models.ApplicationStatusPending}
		if err := s.applications.Create(ctx, application); err != nil {
			return nil, fmt.Errorf("failed to create trainer application: %w", err)
		}
	default:
		return nil, fmt.Errorf("failed to check trainer application: %w", err)
	}

	if strings.TrimSpace(reason) == "" {
		reason = "Promoted directly by an administrator."
	}
	return s.ApproveApplication(ctx, application.ID, adminID, reason)
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func validateDocumentSet(documents []SubmittedDocument) error {
	if len(documents) > MaxSubmittedDocuments {
		return &FormError{Fields: []string{"documents"}}
	}
	certifications, identifications := 0, 0
	for _, document := range documents {
		if err := validateDocumentKind(document.Kind); err != nil {
			return &FormError{Fields: []string{"documents"}}
		}
		if len(document.Data) == 0 {
			return &FormError{Fields: []string{"documents"}}
		}
		switch document.Kind {
		case models.DocumentKindCertification:
			certifications++
		case models.DocumentKindIdentification:
			identifications++
		}
	}
	if certifications == 0 || identifications == 0 {
		return ErrMissingDocuments
	}
	return nil
}

// validateForm checks every field and returns the application with its
// details filled in (no id, user or status yet).
func validateForm(form ApplicationForm, now time.Time) (*models.TrainerApplication, error) {
	var invalid []string
	text := func(field, value string, min, max int) string {
		value = strings.Join(strings.Fields(value), " ")
		length := utf8.RuneCountInString(value)
		if length < min || length > max || hasControl(value) {
			invalid = append(invalid, field)
		}
		return value
	}
	fullName := text("full_name", form.FullName, 3, 120)
	city := text("city", form.City, 2, 80)
	profession := text("profession", form.Profession, 2, 120)

	birth, err := time.Parse("2006-01-02", strings.TrimSpace(form.BirthDate))
	if err != nil || birth.After(now) || age(birth, now) < minimumAge || age(birth, now) > 100 {
		invalid = append(invalid, "birth_date")
	}
	if form.YearsExperience < 0 || form.YearsExperience > 60 {
		invalid = append(invalid, "years_experience")
	}

	if len(form.WorkHistory) == 0 || len(form.WorkHistory) > maxWorkEntries {
		invalid = append(invalid, "work_history")
	}
	work := make([]WorkEntry, 0, len(form.WorkHistory))
	for _, entry := range form.WorkHistory {
		organization := strings.Join(strings.Fields(entry.Organization), " ")
		role := strings.Join(strings.Fields(entry.Role), " ")
		badEnd := entry.EndYear != nil && (*entry.EndYear < entry.StartYear || *entry.EndYear > now.Year())
		if utf8.RuneCountInString(organization) < 2 || utf8.RuneCountInString(organization) > 120 || utf8.RuneCountInString(role) < 2 ||
			utf8.RuneCountInString(role) > 120 || entry.StartYear < 1950 || entry.StartYear > now.Year() || badEnd || hasControl(organization+role) {
			invalid = append(invalid, "work_history")
			break
		}
		work = append(work, WorkEntry{Organization: organization, Role: role, StartYear: entry.StartYear, EndYear: entry.EndYear})
	}

	if len(form.Education) == 0 || len(form.Education) > maxEducationEntries {
		invalid = append(invalid, "education")
	}
	education := make([]EducationEntry, 0, len(form.Education))
	for _, entry := range form.Education {
		institution := strings.Join(strings.Fields(entry.Institution), " ")
		qualification := strings.Join(strings.Fields(entry.Qualification), " ")
		if utf8.RuneCountInString(institution) < 2 || utf8.RuneCountInString(institution) > 120 || utf8.RuneCountInString(qualification) < 2 ||
			utf8.RuneCountInString(qualification) > 120 || entry.Year < 1950 || entry.Year > now.Year()+1 || hasControl(institution+qualification) {
			invalid = append(invalid, "education")
			break
		}
		education = append(education, EducationEntry{Institution: institution, Qualification: qualification, Year: entry.Year})
	}

	about := strings.TrimSpace(form.About)
	if utf8.RuneCountInString(about) < minAbout || utf8.RuneCountInString(about) > maxAbout {
		invalid = append(invalid, "about")
	}

	if len(invalid) > 0 {
		return nil, &FormError{Fields: dedupe(invalid)}
	}
	workJSON, _ := json.Marshal(work)
	educationJSON, _ := json.Marshal(education)
	years := form.YearsExperience
	return &models.TrainerApplication{
		FullName:        &fullName,
		BirthDate:       &birth,
		City:            &city,
		Profession:      &profession,
		YearsExperience: &years,
		WorkHistory:     workJSON,
		Education:       educationJSON,
		About:           &about,
	}, nil
}

func age(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.YearDay() < birth.YearDay() {
		years--
	}
	return years
}

func hasControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return true
		}
	}
	return false
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
