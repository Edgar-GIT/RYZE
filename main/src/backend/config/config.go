package config

import "time"

type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

type JWTConfig struct {
	Secret         string
	AccessTokenTTL time.Duration
	CookieSecure   bool
}

type CORSConfig struct {
	AllowedOrigins []string
}

// Admin1ID and Admin2ID are the identifiers of the configured platform
// administrators. They are the only identities accepted by the admin
// authentication middleware, so the token subject is never trusted as an admin
// identity without passing this check.
const (
	Admin1ID = "ADMIN_1"
	Admin2ID = "ADMIN_2"
)

// IsValidAdminIdentity reports whether id is one of the configured
// administrator identities.
func IsValidAdminIdentity(id string) bool {
	return id == Admin1ID || id == Admin2ID
}

// Admin identifies one configured administrator. Administrators are platform
// accounts defined exclusively through configuration; they are never stored in
// the database.
type Admin struct {
	ID         string
	Username   string
	Password   string
	AccessCode string
}

type AdminConfig struct {
	Admins []Admin
}

// PricingConfig holds the configurable pricing boundaries for program
// products. The minimum price is expressed in minor currency units (cents
// for EUR) so the business can adjust the floor without a code change.
type PricingConfig struct {
	MinProgramPriceMinorUnits int64
}

// CommissionConfig holds the configurable commission parameters for the
// commercial rules system. DefaultPlatformCommissionBPS is the global platform
// commission expressed in basis points (1 bps = 0.01%). When no trainer-specific
// override exists, this default is used for every purchase. The business can
// adjust the default without a code change.
type CommissionConfig struct {
	DefaultPlatformCommissionBPS uint32
}

// StripeConfig holds the Stripe Checkout provider configuration. The provider
// is optional: when the secret key is empty the payment provider falls back to
// the not-configured placeholder. SuccessURL and CancelURL control where the
// user is redirected after the Stripe Checkout flow completes or is cancelled.
type StripeConfig struct {
	SecretKey  string
	SuccessURL string
	CancelURL  string
}

// PayPalConfig holds the PayPal provider configuration. The provider is
// optional: when the client ID is empty the payment provider falls back to the
// not-configured placeholder. Mode must be "sandbox" or "live" to select the
// appropriate PayPal API base URL.
type PayPalConfig struct {
	ClientID string
	Secret   string
	Mode     string
}

// WebhookConfig holds the provider webhook verification configuration. Webhooks
// are the ONLY mechanism that may complete a purchase. Both configurations are
// optional: when empty the corresponding webhook endpoint is not registered.
type WebhookConfig struct {
	StripeWebhookSecret string
	PayPalWebhookID     string
}
