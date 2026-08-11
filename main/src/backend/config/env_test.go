package config_test

import (
	"strings"
	"testing"

	"ryze/backend/config"
)

func TestLoadAdminValidConfig(t *testing.T) {
	t.Setenv("ADMIN_1_USERNAME", "ryzeADMIN1")
	t.Setenv("ADMIN_1_PASSWORD", "edgar_manager123#")
	t.Setenv("ADMIN_2_USERNAME", "ryzeADMIN2")
	t.Setenv("ADMIN_2_PASSWORD", "sandro_manager123#")

	cfg, err := config.LoadAdmin()
	if err != nil {
		t.Fatalf("LoadAdmin: %v", err)
	}
	if len(cfg.Admins) != 2 {
		t.Fatalf("expected 2 admins, got %d", len(cfg.Admins))
	}
	if cfg.Admins[0].ID != "ADMIN_1" || cfg.Admins[0].Username != "ryzeADMIN1" || cfg.Admins[0].Password != "edgar_manager123#" {
		t.Fatalf("unexpected ADMIN_1 config: %+v", cfg.Admins[0])
	}
	if cfg.Admins[1].ID != "ADMIN_2" || cfg.Admins[1].Username != "ryzeADMIN2" || cfg.Admins[1].Password != "sandro_manager123#" {
		t.Fatalf("unexpected ADMIN_2 config: %+v", cfg.Admins[1])
	}
}

func TestLoadAdminMissingUsername(t *testing.T) {
	t.Setenv("ADMIN_1_USERNAME", "")
	t.Setenv("ADMIN_1_PASSWORD", "something123#")
	t.Setenv("ADMIN_2_USERNAME", "ryzeADMIN2")
	t.Setenv("ADMIN_2_PASSWORD", "sandro_manager123#")

	if _, err := config.LoadAdmin(); err == nil {
		t.Fatal("expected error when ADMIN_1_USERNAME is missing")
	} else if !strings.Contains(err.Error(), "ADMIN_1_USERNAME") {
		t.Fatalf("error must mention the missing key, got %v", err)
	}
}

func TestLoadAdminMissingPassword(t *testing.T) {
	t.Setenv("ADMIN_1_USERNAME", "ryzeADMIN1")
	t.Setenv("ADMIN_1_PASSWORD", "")
	t.Setenv("ADMIN_2_USERNAME", "ryzeADMIN2")
	t.Setenv("ADMIN_2_PASSWORD", "sandro_manager123#")

	if _, err := config.LoadAdmin(); err == nil {
		t.Fatal("expected error when ADMIN_1_PASSWORD is missing")
	} else if !strings.Contains(err.Error(), "ADMIN_1_PASSWORD") {
		t.Fatalf("error must mention the missing key, got %v", err)
	}
}

func TestLoadAdminShortPassword(t *testing.T) {
	t.Setenv("ADMIN_1_USERNAME", "ryzeADMIN1")
	t.Setenv("ADMIN_1_PASSWORD", "short")
	t.Setenv("ADMIN_2_USERNAME", "ryzeADMIN2")
	t.Setenv("ADMIN_2_PASSWORD", "sandro_manager123#")

	if _, err := config.LoadAdmin(); err == nil {
		t.Fatal("expected error when an admin password is too short")
	}
}

func TestLoadAdminDuplicateUsernames(t *testing.T) {
	t.Setenv("ADMIN_1_USERNAME", "ryzeADMIN1")
	t.Setenv("ADMIN_1_PASSWORD", "edgar_manager123#")
	t.Setenv("ADMIN_2_USERNAME", "ryzeADMIN1")
	t.Setenv("ADMIN_2_PASSWORD", "sandro_manager123#")

	if _, err := config.LoadAdmin(); err == nil {
		t.Fatal("expected error when two admins share a username")
	}
}

func TestLoadAdminTrimsUsernames(t *testing.T) {
	t.Setenv("ADMIN_1_USERNAME", "  ryzeADMIN1  ")
	t.Setenv("ADMIN_1_PASSWORD", "edgar_manager123#")
	t.Setenv("ADMIN_2_USERNAME", "ryzeADMIN2")
	t.Setenv("ADMIN_2_PASSWORD", "sandro_manager123#")

	cfg, err := config.LoadAdmin()
	if err != nil {
		t.Fatalf("LoadAdmin: %v", err)
	}
	if cfg.Admins[0].Username != "ryzeADMIN1" {
		t.Fatalf("expected trimmed username, got %q", cfg.Admins[0].Username)
	}
}

func TestLoadAdminDoesNotTrimPasswords(t *testing.T) {
	t.Setenv("ADMIN_1_USERNAME", "ryzeADMIN1")
	t.Setenv("ADMIN_1_PASSWORD", "edgar_manager123#")
	t.Setenv("ADMIN_2_USERNAME", "ryzeADMIN2")
	t.Setenv("ADMIN_2_PASSWORD", "sandro_manager123#")

	cfg, err := config.LoadAdmin()
	if err != nil {
		t.Fatalf("LoadAdmin: %v", err)
	}
	if cfg.Admins[0].Password != "edgar_manager123#" {
		t.Fatalf("password must be preserved verbatim, got %q", cfg.Admins[0].Password)
	}
}
