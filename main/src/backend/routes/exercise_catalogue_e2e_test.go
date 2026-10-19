package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"ryze/backend/api/auth"
	"ryze/backend/config"
)

// TestExerciseCatalogueAdminE2E exercises the catalogue administration journey
// through the complete router: the Management Administrator creates a draft,
// the draft stays invisible to the public catalogue, publication makes it
// visible with its structured metadata, archiving hides it again, and the
// Technical Administrator is refused because catalogue curation is a
// Management responsibility.
func TestExerciseCatalogueAdminE2E(t *testing.T) {
	router, _, adminCfg := newE2ERouter(t)

	admin1 := adminByID(t, adminCfg, config.Admin1ID)
	admin2 := adminByID(t, adminCfg, config.Admin2ID)
	admin2Cookie := e2eAdminLogin(t, router, admin2)

	name := fmt.Sprintf("E2E-Catalogue-%d", time.Now().UnixNano())

	// A draft is created through the administrative surface.
	createBody := fmt.Sprintf(`{
		"name": %q,
		"description": "Created by the end-to-end test.",
		"instruction_steps": ["Set up.", "Execute."],
		"body_part": "chest",
		"target_muscles": "chest",
		"difficulty": "Beginner",
		"movement_category": "Compound"
	}`, name)
	rec, raw := e2eRequest(router, auth.AdminAccessTokenCookieName, admin2Cookie, http.MethodPost, "/api/v1/admin/exercises", createBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create exercise: expected 201, got %d (body: %s)", rec.Code, raw)
	}
	var created struct {
		Data struct {
			Exercise struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"exercise"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &created); err != nil {
		t.Fatalf("create exercise: invalid JSON: %v (body: %s)", err, raw)
	}
	if created.Data.Exercise.ID == "" || created.Data.Exercise.Status != "draft" {
		t.Fatalf("create exercise: expected a draft with an id, got %s", raw)
	}
	exerciseID := created.Data.Exercise.ID

	// The draft is invisible in the public catalogue.
	if total := e2ePublicSearchTotal(t, router, name); total != 0 {
		t.Fatalf("draft visibility: expected 0 public matches, got %d", total)
	}

	// Publication makes the entry visible with its structured metadata and
	// media still unavailable because exposure is disabled in the test config.
	rec, raw = e2eRequest(router, auth.AdminAccessTokenCookieName, admin2Cookie, http.MethodPatch,
		"/api/v1/admin/exercises/"+exerciseID, `{"status": "published"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("publish exercise: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !e2ePublicExerciseVisible(t, router, name) {
		t.Fatal("publish exercise: the published entry must be visible publicly")
	}

	// Archiving hides the entry from the public catalogue while the
	// administrative surface still serves it.
	rec, raw = e2eRequest(router, auth.AdminAccessTokenCookieName, admin2Cookie, http.MethodPatch,
		"/api/v1/admin/exercises/"+exerciseID, `{"status": "archived"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("archive exercise: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if total := e2ePublicSearchTotal(t, router, name); total != 0 {
		t.Fatalf("archived visibility: expected 0 public matches, got %d", total)
	}
	rec, raw = e2eRequest(router, auth.AdminAccessTokenCookieName, admin2Cookie, http.MethodGet,
		"/api/v1/admin/exercises/"+exerciseID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin get archived: expected 200, got %d (body: %s)", rec.Code, raw)
	}

	// The Technical Administrator does not hold the catalogue permission.
	admin1Cookie := e2eAdminLogin(t, router, admin1)
	rec, raw = e2eRequest(router, auth.AdminAccessTokenCookieName, admin1Cookie, http.MethodGet, "/api/v1/admin/exercises", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("technical admin read: expected 403, got %d (body: %s)", rec.Code, raw)
	}

	// Without an admin session the catalogue surface stays closed.
	rec, raw = e2eRequest(router, "", "", http.MethodGet, "/api/v1/admin/exercises", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous read: expected 401, got %d (body: %s)", rec.Code, raw)
	}
}

// e2ePublicSearchTotal returns the number of public search matches for a name.
func e2ePublicSearchTotal(t *testing.T, router http.Handler, name string) int64 {
	t.Helper()
	rec, raw := e2eRequest(router, "", "", http.MethodGet, "/api/v1/exercises/search?q="+name, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("public search: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	var payload struct {
		Data struct {
			Pagination struct {
				Total int64 `json:"total"`
			} `json:"pagination"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("public search: invalid JSON: %v (body: %s)", err, raw)
	}
	return payload.Data.Pagination.Total
}

// e2ePublicExerciseVisible searches the public catalogue and verifies that the
// published entry is served with its metadata and the unavailable media state.
func e2ePublicExerciseVisible(t *testing.T, router http.Handler, name string) bool {
	t.Helper()
	rec, raw := e2eRequest(router, "", "", http.MethodGet, "/api/v1/exercises/search?q="+name, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("public search: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	var payload struct {
		Data struct {
			Exercises []struct {
				ID               string   `json:"id"`
				BodyPart         string   `json:"body_part"`
				InstructionSteps []string `json:"instruction_steps"`
				MediaAvailable   bool     `json:"media_available"`
			} `json:"exercises"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("public search: invalid JSON: %v (body: %s)", err, raw)
	}
	if len(payload.Data.Exercises) != 1 {
		return false
	}
	exercise := payload.Data.Exercises[0]
	return exercise.BodyPart == "chest" && len(exercise.InstructionSteps) == 2 && !exercise.MediaAvailable
}
