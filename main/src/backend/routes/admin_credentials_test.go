package routes

import (
	"testing"

	"ryze/backend/config"
)

func TestAdminCredentialsCarryAccessCodes(t *testing.T) {
	cfg := config.AdminConfig{
		Admins: []config.Admin{
			{
				ID:         "ADMIN_1",
				Username:   "ryzeADMIN1",
				Password:   "supersecret",
				AccessCode: "first-factor-two",
			},
			{
				ID:         "ADMIN_2",
				Username:   "ryzeADMIN2",
				Password:   "supersecret",
				AccessCode: "second-factor-two",
			},
		},
	}

	credentials := adminCredentials(cfg)
	if len(credentials) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(credentials))
	}

	for _, credential := range credentials {
		if credential.ID == "" || credential.Username == "" || credential.Password == "" {
			t.Errorf("credential %q must carry id, username and password", credential.ID)
		}
		if credential.AccessCode == "" {
			t.Errorf("credential %q must carry the access code", credential.ID)
		}
	}
}

func TestAdminCredentialsEmpty(t *testing.T) {
	if got := adminCredentials(config.AdminConfig{}); len(got) != 0 {
		t.Fatalf("expected no credentials, got %d", len(got))
	}
}
