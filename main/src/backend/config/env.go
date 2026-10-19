package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/joho/godotenv"

	"ryze/backend/services/password"
)

const (
	defaultHost                             = "127.0.0.1"
	defaultPort                             = "3306"
	defaultTokenTTL                         = 15 * time.Minute
	minJWTSecretLength                      = 32
	maxTokenTTL                             = 24 * time.Hour
	defaultAllowedOrigin                    = "http://localhost:5173,http://127.0.0.1:5173"
	minAdminPasswordLen                     = password.MinPasswordLength
	minAdminAccessCodeLen                   = 8
	defaultRateLimitEnabled                 = true
	defaultRateLimitMaxKeys                 = 10000
	defaultMinProgramPriceMinorUnits        = 100
	defaultPlatformCommissionBPS     uint32 = 2000
	defaultReadHeaderTimeout                = 10 * time.Second
	defaultReadTimeout                      = 30 * time.Second
	defaultWriteTimeout                     = 60 * time.Second
	defaultIdleTimeout                      = 120 * time.Second
	defaultShutdownTimeout                  = 20 * time.Second
	defaultMaxBodyBytes                     = 1 << 20
	defaultExerciseMediaBasePath            = "/media/exercises"
	defaultEvidenceStorageDir               = "storage/trainer-evidence"
	defaultEvidenceMaxBytes                 = 10 << 20
	maxEvidenceMaxBytes                     = 25 << 20
)

// LoadEnvFile loads the repository .env file so environment variables defined
// there are available to the process. It searches the working directory first
// and then walks upwards until the repository root (identified by .git) is found.
func LoadEnvFile() {
	_ = godotenv.Load()

	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			_ = godotenv.Load(filepath.Join(dir, ".env"))
			return
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

// Load reads the database configuration from environment variables.
// DB_HOST and DB_PORT have defaults; DB_NAME and DB_USER are required.
func Load() (DatabaseConfig, error) {
	cfg := DatabaseConfig{
		Host:     valueOr("DB_HOST", defaultHost),
		Port:     valueOr("DB_PORT", defaultPort),
		Name:     os.Getenv("DB_NAME"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
	}

	if cfg.Name == "" {
		return DatabaseConfig{}, fmt.Errorf("DB_NAME is required")
	}
	if cfg.User == "" {
		return DatabaseConfig{}, fmt.Errorf("DB_USER is required")
	}
	if port, err := strconv.Atoi(cfg.Port); err != nil || port < 1 || port > 65535 {
		return DatabaseConfig{}, fmt.Errorf("DB_PORT must be a valid port number, got %q", cfg.Port)
	}

	return cfg, nil
}

func valueOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// LoadJWT reads the JWT configuration from environment variables. The signing
// secret is required and must be strong enough (minimum length) so the server
// never silently falls back to an insecure default. The access-token lifetime
// is optional and defaults to 15 minutes when not provided. The auth cookie is
// Secure by default; local HTTP development may disable it with COOKIE_SECURE.
func LoadJWT() (JWTConfig, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(secret) < minJWTSecretLength {
		return JWTConfig{}, fmt.Errorf("JWT_SECRET must be set and at least %d characters long", minJWTSecretLength)
	}

	ttl := defaultTokenTTL
	if raw := strings.TrimSpace(os.Getenv("JWT_ACCESS_TOKEN_TTL")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			return JWTConfig{}, fmt.Errorf("JWT_ACCESS_TOKEN_TTL must be a positive duration, got %q", raw)
		}
		// A session token outlives every revocation shortcut, so its lifetime
		// is capped: a misconfigured days-long TTL would silently weaken the
		// session model. Longer sessions must be an explicit future decision.
		if parsed > maxTokenTTL {
			return JWTConfig{}, fmt.Errorf("JWT_ACCESS_TOKEN_TTL must not exceed %s, got %q", maxTokenTTL, raw)
		}
		ttl = parsed
	}

	cookieSecure := true
	if raw := strings.TrimSpace(os.Getenv("COOKIE_SECURE")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return JWTConfig{}, fmt.Errorf("COOKIE_SECURE must be a boolean, got %q", raw)
		}
		cookieSecure = parsed
	}

	return JWTConfig{Secret: secret, AccessTokenTTL: ttl, CookieSecure: cookieSecure}, nil
}

// LoadAdmin reads the configured administrators from environment variables.
// Every admin requires a username, a password and an access code (the second
// authentication factor) of at least their minimum length; missing or invalid
// values fail at startup instead of silently disabling an administrator.
// Usernames must be unique so credentials always resolve to a single admin
// identity.
func LoadAdmin() (AdminConfig, error) {
	admins := []Admin{
		{ID: Admin1ID},
		{ID: Admin2ID},
	}

	seenUsernames := map[string]string{}
	for i, admin := range admins {
		username := strings.TrimSpace(os.Getenv(admin.ID + "_USERNAME"))
		password := os.Getenv(admin.ID + "_PASSWORD")
		accessCode := os.Getenv(admin.ID + "_ACCESS_CODE")
		if username == "" {
			return AdminConfig{}, fmt.Errorf("%s_USERNAME is required", admin.ID)
		}
		if utf8.RuneCountInString(password) < minAdminPasswordLen {
			return AdminConfig{}, fmt.Errorf("%s_PASSWORD is required and must be at least %d characters long", admin.ID, minAdminPasswordLen)
		}
		if len(accessCode) < minAdminAccessCodeLen {
			return AdminConfig{}, fmt.Errorf("%s_ACCESS_CODE is required and must be at least %d characters long", admin.ID, minAdminAccessCodeLen)
		}
		if previous, exists := seenUsernames[username]; exists {
			return AdminConfig{}, fmt.Errorf("admin username %q is already used by %s and %s", username, previous, admin.ID)
		}
		seenUsernames[username] = admin.ID
		admins[i].Username = username
		admins[i].Password = password
		admins[i].AccessCode = accessCode
	}

	return AdminConfig{Admins: admins}, nil
}

// LoadCORS reads the cross-origin configuration from environment variables.
// CORS_ALLOWED_ORIGINS is a comma-separated list of origins allowed to call the
// API with credentials. When unset it defaults to the local Vite development
// origins so the frontend can talk to the backend without extra setup.
func LoadCORS() (CORSConfig, error) {
	origins := []string{}
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		raw = defaultAllowedOrigin
	}

	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			return CORSConfig{}, fmt.Errorf("CORS_ALLOWED_ORIGINS contains an empty origin")
		}
		origins = append(origins, origin)
	}

	return CORSConfig{AllowedOrigins: origins}, nil
}

// LoadPricing reads the pricing configuration from environment variables.
// MIN_PROGRAM_PRICE_MINOR_UNITS sets the minimum price in minor currency units
// (cents for EUR) that a paid program may have. When unset it defaults to 100
// (€1.00). Free programs always have a price of 0 regardless of this setting.
func LoadPricing() (PricingConfig, error) {
	minPrice := int64(defaultMinProgramPriceMinorUnits)
	if raw := strings.TrimSpace(os.Getenv("MIN_PROGRAM_PRICE_MINOR_UNITS")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			return PricingConfig{}, fmt.Errorf("MIN_PROGRAM_PRICE_MINOR_UNITS must be a non-negative integer, got %q", raw)
		}
		minPrice = parsed
	}

	return PricingConfig{MinProgramPriceMinorUnits: minPrice}, nil
}

// LoadCommission reads the commission configuration from environment variables.
// DEFAULT_PLATFORM_COMMISSION_BPS sets the global platform commission in basis
// points (1 bps = 0.01%). When unset it defaults to 2000 (20%). Values above
// 10000 (100%) are rejected because a platform commission cannot exceed the
// full price.
func LoadCommission() (CommissionConfig, error) {
	bps := defaultPlatformCommissionBPS
	if raw := strings.TrimSpace(os.Getenv("DEFAULT_PLATFORM_COMMISSION_BPS")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || parsed > 10000 {
			return CommissionConfig{}, fmt.Errorf("DEFAULT_PLATFORM_COMMISSION_BPS must be a non-negative integer no greater than 10000, got %q", raw)
		}
		bps = uint32(parsed)
	}

	return CommissionConfig{DefaultPlatformCommissionBPS: bps}, nil
}

// LoadStripe reads the Stripe Checkout provider configuration from environment
// variables. All values are optional: when STRIPE_SECRET_KEY is empty the
// payment provider falls back to the not-configured placeholder. SuccessURL and
// CancelURL default to empty strings, which causes the Stripe SDK to use the
// default redirect behaviour.
func LoadStripe() StripeConfig {
	return StripeConfig{
		SecretKey:  strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY")),
		SuccessURL: strings.TrimSpace(os.Getenv("STRIPE_SUCCESS_URL")),
		CancelURL:  strings.TrimSpace(os.Getenv("STRIPE_CANCEL_URL")),
	}
}

// LoadPayPal reads the PayPal provider configuration from environment
// variables. All values are optional: when PAYPAL_CLIENT_ID is empty the
// payment provider falls back to the not-configured placeholder. Mode defaults
// to "sandbox" when not provided. PAYPAL_RETURN_URL and PAYPAL_CANCEL_URL are
// the redirect templates used for the PayPal checkout flow; when empty the
// corresponding redirect is not configured on the created orders.
func LoadPayPal() PayPalConfig {
	mode := strings.TrimSpace(os.Getenv("PAYPAL_MODE"))
	if mode == "" {
		mode = "sandbox"
	}

	return PayPalConfig{
		ClientID:  strings.TrimSpace(os.Getenv("PAYPAL_CLIENT_ID")),
		Secret:    strings.TrimSpace(os.Getenv("PAYPAL_SECRET")),
		Mode:      mode,
		ReturnURL: strings.TrimSpace(os.Getenv("PAYPAL_RETURN_URL")),
		CancelURL: strings.TrimSpace(os.Getenv("PAYPAL_CANCEL_URL")),
	}
}

// LoadWebhooks reads the webhook verification configuration from environment
// variables. Both values are optional: when empty the corresponding webhook
// endpoint is not registered. STRIPE_WEBHOOK_SECRET is the Stripe webhook
// signing secret used to verify incoming Stripe webhook signatures.
// PAYPAL_WEBHOOK_ID is the PayPal webhook ID used to verify incoming PayPal
// webhook authenticity.
func LoadWebhooks() WebhookConfig {
	return WebhookConfig{
		StripeWebhookSecret: strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET")),
		PayPalWebhookID:     strings.TrimSpace(os.Getenv("PAYPAL_WEBHOOK_ID")),
	}
}

// LoadTestMode reads the Test Mode configuration from environment variables.
// TEST_MODE_ENABLED gates the feature globally; it is disabled by default and
// only "true" enables it. Invalid boolean values fail at startup so a typo can
// never silently leave the intended policy unset.
func LoadTestMode() (TestModeConfig, error) {
	raw := strings.TrimSpace(os.Getenv("TEST_MODE_ENABLED"))
	if raw == "" {
		return TestModeConfig{Enabled: false}, nil
	}
	enabled, err := strconv.ParseBool(raw)
	if err != nil {
		return TestModeConfig{}, fmt.Errorf("TEST_MODE_ENABLED must be a boolean, got %q", raw)
	}
	return TestModeConfig{Enabled: enabled}, nil
}

// LoadExerciseMedia reads the exercise-media exposure configuration from
// environment variables. EXERCISE_MEDIA_ENABLED gates the whole surface; it is
// disabled by default because the bundled dataset media licensing must be
// confirmed before any public or production delivery (see security.MD). When
// enabled, EXERCISE_MEDIA_BASE_PATH is the same-origin path the API prefixes to
// stored relative media paths; a value containing a scheme or parent traversal
// is rejected at startup.
func LoadExerciseMedia() (ExerciseMediaConfig, error) {
	cfg := ExerciseMediaConfig{
		Enabled:  false,
		BasePath: defaultExerciseMediaBasePath,
	}

	if raw := strings.TrimSpace(os.Getenv("EXERCISE_MEDIA_ENABLED")); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return ExerciseMediaConfig{}, fmt.Errorf("EXERCISE_MEDIA_ENABLED must be a boolean, got %q", raw)
		}
		cfg.Enabled = enabled
	}

	if raw := strings.TrimSpace(os.Getenv("EXERCISE_MEDIA_BASE_PATH")); raw != "" {
		cfg.BasePath = raw
	}
	cfg.BasePath = strings.TrimSuffix(cfg.BasePath, "/")
	// A base path must be a same-origin absolute path: a scheme, a
	// protocol-relative prefix ("//host") or parent traversal would let the
	// API hand clients a URL that escapes the RYZE origin.
	if !strings.HasPrefix(cfg.BasePath, "/") || strings.HasPrefix(cfg.BasePath, "//") || strings.Contains(cfg.BasePath, "..") || strings.Contains(cfg.BasePath, "://") {
		return ExerciseMediaConfig{}, fmt.Errorf("EXERCISE_MEDIA_BASE_PATH must be a same-origin absolute path without parent traversal, got %q", cfg.BasePath)
	}

	return cfg, nil
}

// LoadEvidence reads the trainer-application evidence store configuration from
// environment variables. TRAINER_EVIDENCE_DIR defaults to a private directory
// outside any web root; a value must be non-empty and free of NUL bytes.
// TRAINER_EVIDENCE_MAX_BYTES bounds one stored document and is clamped to the
// interval [1 byte, 25 MiB]: values above the maximum are rejected at startup so
// a typo can never raise the bound beyond what the upload route and the
// deployment are prepared to handle.
func LoadEvidence() (EvidenceConfig, error) {
	cfg := EvidenceConfig{
		StorageDir: defaultEvidenceStorageDir,
		MaxBytes:   defaultEvidenceMaxBytes,
	}

	if raw := os.Getenv("TRAINER_EVIDENCE_DIR"); strings.TrimSpace(raw) != "" {
		if strings.ContainsRune(raw, 0) {
			return EvidenceConfig{}, fmt.Errorf("TRAINER_EVIDENCE_DIR contains an invalid character")
		}
		cfg.StorageDir = strings.TrimSpace(raw)
	}

	if raw := strings.TrimSpace(os.Getenv("TRAINER_EVIDENCE_MAX_BYTES")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			return EvidenceConfig{}, fmt.Errorf("TRAINER_EVIDENCE_MAX_BYTES must be a positive integer, got %q", raw)
		}
		if parsed > maxEvidenceMaxBytes {
			return EvidenceConfig{}, fmt.Errorf("TRAINER_EVIDENCE_MAX_BYTES must not exceed %d bytes, got %q", maxEvidenceMaxBytes, raw)
		}
		cfg.MaxBytes = parsed
	}

	return cfg, nil
}

// LoadServer reads the HTTP server safety configuration from environment
// variables. Every value is optional and falls back to a safe default; a
// present-but-invalid value fails at startup so a typo can never silently leave
// a timeout disabled or the body limit unbounded.
func LoadServer() (ServerConfig, error) {
	cfg := ServerConfig{
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		ShutdownTimeout:   defaultShutdownTimeout,
		MaxBodyBytes:      defaultMaxBodyBytes,
	}

	var err error
	if cfg.ReadHeaderTimeout, err = durationOr("SERVER_READ_HEADER_TIMEOUT", cfg.ReadHeaderTimeout); err != nil {
		return ServerConfig{}, err
	}
	if cfg.ReadTimeout, err = durationOr("SERVER_READ_TIMEOUT", cfg.ReadTimeout); err != nil {
		return ServerConfig{}, err
	}
	if cfg.WriteTimeout, err = durationOr("SERVER_WRITE_TIMEOUT", cfg.WriteTimeout); err != nil {
		return ServerConfig{}, err
	}
	if cfg.IdleTimeout, err = durationOr("SERVER_IDLE_TIMEOUT", cfg.IdleTimeout); err != nil {
		return ServerConfig{}, err
	}
	if cfg.ShutdownTimeout, err = durationOr("SERVER_SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout); err != nil {
		return ServerConfig{}, err
	}

	if raw := strings.TrimSpace(os.Getenv("SERVER_MAX_BODY_BYTES")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			return ServerConfig{}, fmt.Errorf("SERVER_MAX_BODY_BYTES must be a positive integer, got %q", raw)
		}
		cfg.MaxBodyBytes = parsed
	}

	return cfg, nil
}

// LoadRateLimit reads the authentication rate-limit configuration from
// environment variables. It is disabled only through an explicit value,
// trusted proxies default to none so the client address can never be spoofed
// through forwarded headers, and an entry that is neither an IP nor a CIDR
// range fails at startup instead of silently being ignored.
func LoadRateLimit() (RateLimitConfig, error) {
	cfg := RateLimitConfig{
		Enabled: defaultRateLimitEnabled,
		MaxKeys: defaultRateLimitMaxKeys,
	}

	if raw := strings.TrimSpace(os.Getenv("RATE_LIMIT_ENABLED")); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return RateLimitConfig{}, fmt.Errorf("RATE_LIMIT_ENABLED must be a boolean, got %q", raw)
		}
		cfg.Enabled = enabled
	}

	if raw := strings.TrimSpace(os.Getenv("RATE_LIMIT_MAX_KEYS")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return RateLimitConfig{}, fmt.Errorf("RATE_LIMIT_MAX_KEYS must be a positive integer, got %q", raw)
		}
		cfg.MaxKeys = parsed
	}

	if raw := os.Getenv("RATE_LIMIT_TRUSTED_PROXIES"); strings.TrimSpace(raw) != "" {
		for _, entry := range strings.Split(raw, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			if !isTrustedProxyEntry(entry) {
				return RateLimitConfig{}, fmt.Errorf("RATE_LIMIT_TRUSTED_PROXIES entry %q is not a valid trusted proxy: a bare IP address or a CIDR range narrower than /0 is required", entry)
			}
			cfg.TrustedProxies = append(cfg.TrustedProxies, entry)
		}
	}

	return cfg, nil
}

// isTrustedProxyEntry reports whether a configured proxy entry is a valid IPv4
// or IPv6 address or a valid CIDR range that is narrower than /0. A /0 range
// trusts every address and would let any client spoof its address through
// forwarded headers, so it is rejected like any other invalid entry.
func isTrustedProxyEntry(entry string) bool {
	if _, network, err := net.ParseCIDR(entry); err == nil {
		ones, _ := network.Mask.Size()
		return ones > 0
	}
	return net.ParseIP(entry) != nil
}

// durationOr parses a positive duration from the environment. It returns the
// fallback when the variable is unset and an error when it is present but
// invalid or non-positive.
func durationOr(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration, got %q", key, raw)
	}
	return parsed, nil
}
