package config_test

import (
	"strings"
	"testing"
	"time"

	"ryze/backend/config"
)

const (
	admin1Username   = "ryzeADMIN1"
	admin1Password   = "edgar_manager123#"
	admin1AccessCode = "mz7Qx2!ryze2fa_admin1"
	admin2Username   = "ryzeADMIN2"
	admin2Password   = "sandro_manager123#"
	admin2AccessCode = "np9Wv4#ryze2fa_admin2"
)

// setAdminEnv sets every admin variable, so tests can clear or replace single
// keys to exercise validation.
func setAdminEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ADMIN_1_USERNAME", admin1Username)
	t.Setenv("ADMIN_1_PASSWORD", admin1Password)
	t.Setenv("ADMIN_1_ACCESS_CODE", admin1AccessCode)
	t.Setenv("ADMIN_2_USERNAME", admin2Username)
	t.Setenv("ADMIN_2_PASSWORD", admin2Password)
	t.Setenv("ADMIN_2_ACCESS_CODE", admin2AccessCode)
}

func TestLoadAdminValidConfig(t *testing.T) {
	setAdminEnv(t)

	cfg, err := config.LoadAdmin()
	if err != nil {
		t.Fatalf("LoadAdmin: %v", err)
	}
	if len(cfg.Admins) != 2 {
		t.Fatalf("expected 2 admins, got %d", len(cfg.Admins))
	}
	if cfg.Admins[0].ID != "ADMIN_1" || cfg.Admins[0].Username != admin1Username || cfg.Admins[0].Password != admin1Password || cfg.Admins[0].AccessCode != admin1AccessCode {
		t.Fatalf("unexpected ADMIN_1 config: %+v", cfg.Admins[0])
	}
	if cfg.Admins[1].ID != "ADMIN_2" || cfg.Admins[1].Username != admin2Username || cfg.Admins[1].Password != admin2Password || cfg.Admins[1].AccessCode != admin2AccessCode {
		t.Fatalf("unexpected ADMIN_2 config: %+v", cfg.Admins[1])
	}
}

func TestLoadAdminMissingAccessCode(t *testing.T) {
	setAdminEnv(t)
	t.Setenv("ADMIN_1_ACCESS_CODE", "")

	if _, err := config.LoadAdmin(); err == nil {
		t.Fatal("expected error when ADMIN_1_ACCESS_CODE is missing")
	} else if !strings.Contains(err.Error(), "ADMIN_1_ACCESS_CODE") {
		t.Fatalf("error must mention the missing key, got %v", err)
	}
}

func TestLoadAdminShortAccessCode(t *testing.T) {
	setAdminEnv(t)
	t.Setenv("ADMIN_1_ACCESS_CODE", "short")

	if _, err := config.LoadAdmin(); err == nil {
		t.Fatal("expected error when an admin access code is too short")
	}
}

func TestLoadAdminMissingUsername(t *testing.T) {
	setAdminEnv(t)
	t.Setenv("ADMIN_1_USERNAME", "")

	if _, err := config.LoadAdmin(); err == nil {
		t.Fatal("expected error when ADMIN_1_USERNAME is missing")
	} else if !strings.Contains(err.Error(), "ADMIN_1_USERNAME") {
		t.Fatalf("error must mention the missing key, got %v", err)
	}
}

func TestLoadAdminMissingPassword(t *testing.T) {
	setAdminEnv(t)
	t.Setenv("ADMIN_1_PASSWORD", "")

	if _, err := config.LoadAdmin(); err == nil {
		t.Fatal("expected error when ADMIN_1_PASSWORD is missing")
	} else if !strings.Contains(err.Error(), "ADMIN_1_PASSWORD") {
		t.Fatalf("error must mention the missing key, got %v", err)
	}
}

func TestLoadAdminShortPassword(t *testing.T) {
	setAdminEnv(t)
	t.Setenv("ADMIN_1_PASSWORD", "short")

	if _, err := config.LoadAdmin(); err == nil {
		t.Fatal("expected error when an admin password is too short")
	}
}

func TestLoadAdminDuplicateUsernames(t *testing.T) {
	setAdminEnv(t)
	t.Setenv("ADMIN_2_USERNAME", admin1Username)

	if _, err := config.LoadAdmin(); err == nil {
		t.Fatal("expected error when two admins share a username")
	}
}

func TestLoadAdminTrimsUsernames(t *testing.T) {
	setAdminEnv(t)
	t.Setenv("ADMIN_1_USERNAME", "  "+admin1Username+"  ")

	cfg, err := config.LoadAdmin()
	if err != nil {
		t.Fatalf("LoadAdmin: %v", err)
	}
	if cfg.Admins[0].Username != admin1Username {
		t.Fatalf("expected trimmed username, got %q", cfg.Admins[0].Username)
	}
}

func TestLoadAdminDoesNotTrimPasswords(t *testing.T) {
	setAdminEnv(t)

	cfg, err := config.LoadAdmin()
	if err != nil {
		t.Fatalf("LoadAdmin: %v", err)
	}
	if cfg.Admins[0].Password != admin1Password {
		t.Fatalf("password must be preserved verbatim, got %q", cfg.Admins[0].Password)
	}
}

func TestLoadPricingDefault(t *testing.T) {
	cfg, err := config.LoadPricing()
	if err != nil {
		t.Fatalf("LoadPricing: %v", err)
	}
	if cfg.MinProgramPriceMinorUnits != 100 {
		t.Fatalf("expected default min price 100, got %d", cfg.MinProgramPriceMinorUnits)
	}
}

func TestLoadPricingCustomValue(t *testing.T) {
	t.Setenv("MIN_PROGRAM_PRICE_MINOR_UNITS", "250")
	cfg, err := config.LoadPricing()
	if err != nil {
		t.Fatalf("LoadPricing: %v", err)
	}
	if cfg.MinProgramPriceMinorUnits != 250 {
		t.Fatalf("expected min price 250, got %d", cfg.MinProgramPriceMinorUnits)
	}
}

func TestLoadPricingZeroAllowed(t *testing.T) {
	t.Setenv("MIN_PROGRAM_PRICE_MINOR_UNITS", "0")
	cfg, err := config.LoadPricing()
	if err != nil {
		t.Fatalf("LoadPricing: %v", err)
	}
	if cfg.MinProgramPriceMinorUnits != 0 {
		t.Fatalf("expected min price 0, got %d", cfg.MinProgramPriceMinorUnits)
	}
}

func TestLoadPricingRejectsNegative(t *testing.T) {
	t.Setenv("MIN_PROGRAM_PRICE_MINOR_UNITS", "-1")
	_, err := config.LoadPricing()
	if err == nil {
		t.Fatal("expected error for negative value")
	}
}

func TestLoadPricingRejectsNonNumeric(t *testing.T) {
	t.Setenv("MIN_PROGRAM_PRICE_MINOR_UNITS", "abc")
	_, err := config.LoadPricing()
	if err == nil {
		t.Fatal("expected error for non-numeric value")
	}
}

// clearServerEnv clears every server variable so a test observes the defaults
// regardless of the ambient environment.
func clearServerEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"SERVER_READ_HEADER_TIMEOUT",
		"SERVER_READ_TIMEOUT",
		"SERVER_WRITE_TIMEOUT",
		"SERVER_IDLE_TIMEOUT",
		"SERVER_SHUTDOWN_TIMEOUT",
		"SERVER_MAX_BODY_BYTES",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadServerDefaults(t *testing.T) {
	clearServerEnv(t)

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("LoadServer: %v", err)
	}
	if cfg.ReadHeaderTimeout != 10*time.Second {
		t.Fatalf("expected default read header timeout 10s, got %s", cfg.ReadHeaderTimeout)
	}
	if cfg.ReadTimeout != 30*time.Second {
		t.Fatalf("expected default read timeout 30s, got %s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 60*time.Second {
		t.Fatalf("expected default write timeout 60s, got %s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 120*time.Second {
		t.Fatalf("expected default idle timeout 120s, got %s", cfg.IdleTimeout)
	}
	if cfg.ShutdownTimeout != 20*time.Second {
		t.Fatalf("expected default shutdown timeout 20s, got %s", cfg.ShutdownTimeout)
	}
	if cfg.MaxBodyBytes != 1<<20 {
		t.Fatalf("expected default max body bytes 1 MiB, got %d", cfg.MaxBodyBytes)
	}
}

func TestLoadServerCustomValues(t *testing.T) {
	clearServerEnv(t)
	t.Setenv("SERVER_READ_HEADER_TIMEOUT", "3s")
	t.Setenv("SERVER_READ_TIMEOUT", "7s")
	t.Setenv("SERVER_WRITE_TIMEOUT", "9s")
	t.Setenv("SERVER_IDLE_TIMEOUT", "11s")
	t.Setenv("SERVER_SHUTDOWN_TIMEOUT", "13s")
	t.Setenv("SERVER_MAX_BODY_BYTES", "2048")

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("LoadServer: %v", err)
	}
	if cfg.ReadHeaderTimeout != 3*time.Second || cfg.ReadTimeout != 7*time.Second || cfg.WriteTimeout != 9*time.Second {
		t.Fatalf("unexpected read/write timeouts: %+v", cfg)
	}
	if cfg.IdleTimeout != 11*time.Second || cfg.ShutdownTimeout != 13*time.Second {
		t.Fatalf("unexpected idle/shutdown timeouts: %+v", cfg)
	}
	if cfg.MaxBodyBytes != 2048 {
		t.Fatalf("expected max body bytes 2048, got %d", cfg.MaxBodyBytes)
	}
}

func TestLoadServerRejectsInvalidDuration(t *testing.T) {
	clearServerEnv(t)
	t.Setenv("SERVER_READ_TIMEOUT", "not-a-duration")

	if _, err := config.LoadServer(); err == nil {
		t.Fatal("expected error for an invalid duration")
	}
}

func TestLoadServerRejectsNonPositiveDuration(t *testing.T) {
	clearServerEnv(t)
	t.Setenv("SERVER_WRITE_TIMEOUT", "0s")

	if _, err := config.LoadServer(); err == nil {
		t.Fatal("expected error for a non-positive duration")
	}
}

func TestLoadServerRejectsInvalidBodyBytes(t *testing.T) {
	clearServerEnv(t)
	t.Setenv("SERVER_MAX_BODY_BYTES", "abc")

	if _, err := config.LoadServer(); err == nil {
		t.Fatal("expected error for a non-numeric body limit")
	}
}

func TestLoadServerRejectsNonPositiveBodyBytes(t *testing.T) {
	clearServerEnv(t)
	t.Setenv("SERVER_MAX_BODY_BYTES", "0")

	if _, err := config.LoadServer(); err == nil {
		t.Fatal("expected error for a non-positive body limit")
	}
}
