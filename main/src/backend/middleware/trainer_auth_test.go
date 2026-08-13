package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ryze/backend/middleware"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/middleware/trainercontext"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeTrainerRepository struct {
	findByUserID func(context.Context, string) (*models.Trainer, error)
}

func (f fakeTrainerRepository) Create(context.Context, *models.Trainer) error {
	return errors.New("not implemented")
}

func (f fakeTrainerRepository) FindByID(context.Context, string) (*models.Trainer, error) {
	return nil, errors.New("not implemented")
}

func (f fakeTrainerRepository) FindByIDIncludingDeleted(context.Context, string) (*models.Trainer, error) {
	return nil, errors.New("not implemented")
}

func (f fakeTrainerRepository) FindByUserID(ctx context.Context, userID string) (*models.Trainer, error) {
	if f.findByUserID == nil {
		return nil, errors.New("find by user id not configured")
	}
	return f.findByUserID(ctx, userID)
}

func (f fakeTrainerRepository) ListActive(context.Context, int, int) ([]models.Trainer, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (f fakeTrainerRepository) ListDeleted(context.Context, int, int) ([]models.Trainer, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (f fakeTrainerRepository) SoftDelete(context.Context, string) error {
	return errors.New("not implemented")
}

func (f fakeTrainerRepository) Reactivate(context.Context, string) error {
	return errors.New("not implemented")
}

func request(router http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/trainer/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func setAuthenticatedUser(c *gin.Context, userID string) {
	c.Set(authcontext.UserIDContextKey, userID)
}

func newContextRouter(
	t *testing.T,
	repo repositories.TrainerRepository,
	userID any,
) (*gin.Engine, *bool) {
	t.Helper()

	reached := false
	router := gin.New()

	router.GET("/trainer/protected", func(c *gin.Context) {
		if userID != nil {
			c.Set(authcontext.UserIDContextKey, userID)
		}

		middleware.TrainerAuthenticate(repo)(c)

		if !c.IsAborted() {
			reached = true
		}
	})

	return router, &reached
}

func TestTrainerAuthenticateContinuesForActiveTrainer(t *testing.T) {
	userID := uuid.NewString()
	trainerID := uuid.NewString()

	repo := fakeTrainerRepository{
		findByUserID: func(_ context.Context, gotUserID string) (*models.Trainer, error) {
			if gotUserID != userID {
				t.Fatalf("expected repository user ID %q, got %q", userID, gotUserID)
			}

			return &models.Trainer{
				ID:     trainerID,
				UserID: userID,
			}, nil
		},
	}

	reached := false
	var gotIdentity trainercontext.Identity

	router := gin.New()
	router.GET("/trainer/protected", func(c *gin.Context) {
		setAuthenticatedUser(c, userID)

		middleware.TrainerAuthenticate(repo)(c)

		if c.IsAborted() {
			return
		}

		reached = true
		gotIdentity, _ = trainercontext.IdentityFromContext(c)
		c.Status(http.StatusOK)
	})

	rec := request(router)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if !reached {
		t.Fatal("protected handler must be reached for an active trainer")
	}

	if gotIdentity.UserID != userID {
		t.Fatalf("expected context user ID %q, got %q", userID, gotIdentity.UserID)
	}

	if gotIdentity.TrainerID != trainerID {
		t.Fatalf("expected context trainer ID %q, got %q", trainerID, gotIdentity.TrainerID)
	}
}

func TestTrainerAuthenticateRejectsMissingUserAuthentication(t *testing.T) {
	repo := fakeTrainerRepository{}

	router, reached := newContextRouter(t, repo, nil)

	rec := request(router)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	if *reached {
		t.Fatal("protected handler must not be reached without user authentication")
	}

	assertTrainerAuthenticationError(t, rec.Body.String())
}

func TestTrainerAuthenticateRejectsInvalidUserContext(t *testing.T) {
	tests := map[string]any{
		"wrong type": 42,
		"empty":      "",
		"not uuid":   "not-a-uuid",
	}

	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			repo := fakeTrainerRepository{}

			router, reached := newContextRouter(t, repo, value)

			rec := request(router)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rec.Code)
			}

			if *reached {
				t.Fatal("protected handler must not be reached with invalid user context")
			}

			assertTrainerAuthenticationError(t, rec.Body.String())
		})
	}
}

func TestTrainerAuthenticateRejectsNonTrainer(t *testing.T) {
	repo := fakeTrainerRepository{
		findByUserID: func(context.Context, string) (*models.Trainer, error) {
			return nil, repositories.ErrTrainerNotFound
		},
	}

	router, reached := newContextRouter(t, repo, uuid.NewString())

	rec := request(router)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	if *reached {
		t.Fatal("protected handler must not be reached for a non-trainer")
	}

	assertTrainerForbiddenError(t, rec.Body.String())
}

func TestTrainerAuthenticateRejectsSoftDeletedTrainer(t *testing.T) {
	repo := fakeTrainerRepository{
		findByUserID: func(context.Context, string) (*models.Trainer, error) {
			return nil, repositories.ErrTrainerNotFound
		},
	}

	router, reached := newContextRouter(t, repo, uuid.NewString())

	rec := request(router)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	if *reached {
		t.Fatal("protected handler must not be reached for a soft-deleted trainer")
	}

	assertTrainerForbiddenError(t, rec.Body.String())
}

func TestTrainerAuthenticateInternalRepositoryFailure(t *testing.T) {
	repo := fakeTrainerRepository{
		findByUserID: func(context.Context, string) (*models.Trainer, error) {
			return nil, errors.New(
				"database connection failed: secret internal detail",
			)
		},
	}

	router, reached := newContextRouter(t, repo, uuid.NewString())

	rec := request(router)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	if *reached {
		t.Fatal("protected handler must not be reached after a repository failure")
	}

	body := rec.Body.String()

	if strings.Contains(body, "database connection failed") ||
		strings.Contains(body, "secret internal detail") {
		t.Fatalf("internal repository details leaked: %s", body)
	}

	if !strings.Contains(body, `"success":false`) ||
		!strings.Contains(body, `"code":"INTERNAL_ERROR"`) {
		t.Fatalf("expected generic internal error envelope, got %s", body)
	}
}

func TestTrainerAuthenticateDoesNotAcceptTrainerContextWithoutUserAuthentication(t *testing.T) {
	userID := uuid.NewString()
	trainerID := uuid.NewString()

	repo := fakeTrainerRepository{
		findByUserID: func(context.Context, string) (*models.Trainer, error) {
			return &models.Trainer{
				ID:     trainerID,
				UserID: userID,
			}, nil
		},
	}

	router := gin.New()
	reached := false

	router.GET("/trainer/protected", func(c *gin.Context) {
		trainercontext.SetIdentity(c, trainercontext.Identity{
			UserID:    userID,
			TrainerID: trainerID,
		})

		middleware.TrainerAuthenticate(repo)(c)

		if !c.IsAborted() {
			reached = true
		}
	})

	rec := request(router)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	if reached {
		t.Fatal("trainer context alone must never authenticate a request")
	}

	assertTrainerAuthenticationError(t, rec.Body.String())
}

func TestTrainerAuthenticateContextIdentityIsNotReusedFromIncomingContext(t *testing.T) {
	userID := uuid.NewString()
	trainerID := uuid.NewString()

	otherUserID := uuid.NewString()
	otherTrainerID := uuid.NewString()

	repo := fakeTrainerRepository{
		findByUserID: func(_ context.Context, gotUserID string) (*models.Trainer, error) {
			if gotUserID != userID {
				t.Fatalf(
					"expected repository lookup for authenticated user %q, got %q",
					userID,
					gotUserID,
				)
			}

			return &models.Trainer{
				ID:     trainerID,
				UserID: userID,
			}, nil
		},
	}

	router := gin.New()
	var gotIdentity trainercontext.Identity

	router.GET("/trainer/protected", func(c *gin.Context) {
		setAuthenticatedUser(c, userID)

		trainercontext.SetIdentity(c, trainercontext.Identity{
			UserID:    otherUserID,
			TrainerID: otherTrainerID,
		})

		middleware.TrainerAuthenticate(repo)(c)

		if c.IsAborted() {
			return
		}

		gotIdentity, _ = trainercontext.IdentityFromContext(c)
		c.Status(http.StatusOK)
	})

	rec := request(router)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if gotIdentity.UserID != userID ||
		gotIdentity.TrainerID != trainerID {
		t.Fatalf(
			"middleware must overwrite stale trainer context, got %+v",
			gotIdentity,
		)
	}
}

func assertTrainerAuthenticationError(t *testing.T, body string) {
	t.Helper()

	if !strings.Contains(body, `"success":false`) {
		t.Fatalf("expected success:false, got %s", body)
	}

	if !strings.Contains(body, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", body)
	}

	if strings.Contains(body, "expired") ||
		strings.Contains(body, "signature") ||
		strings.Contains(body, "malformed") {
		t.Fatalf("authentication failure leaked internal details: %s", body)
	}
}

func assertTrainerForbiddenError(t *testing.T, body string) {
	t.Helper()

	if !strings.Contains(body, `"success":false`) {
		t.Fatalf("expected success:false, got %s", body)
	}

	if !strings.Contains(body, `"code":"FORBIDDEN"`) {
		t.Fatalf("expected FORBIDDEN, got %s", body)
	}
}
