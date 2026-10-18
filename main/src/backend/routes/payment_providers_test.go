package routes

import (
	"testing"

	stripe "github.com/stripe/stripe-go/v86"

	"ryze/backend/config"
	"ryze/backend/services/payments"
)

// TestResolvePaymentProviders_StripeRequiresSecretAndWebhookSigningSecret
// verifies that the card method is only enabled when Stripe is completely
// configured. Stripe documents webhooks as required for fulfillment, so a
// secret key without a signing secret could never reliably complete a purchase
// and must never be advertised.
func TestResolvePaymentProviders_StripeRequiresSecretAndWebhookSigningSecret(t *testing.T) {
	cases := []struct {
		name        string
		secretKey   string
		webhookKey  string
		wantEnabled bool
	}{
		{name: "fully configured", secretKey: "sk_test_123", webhookKey: "whsec_123", wantEnabled: true},
		{name: "secret without webhook secret", secretKey: "sk_test_123", webhookKey: "", wantEnabled: false},
		{name: "webhook secret without secret", secretKey: "", webhookKey: "whsec_123", wantEnabled: false},
		{name: "nothing configured", secretKey: "", webhookKey: "", wantEnabled: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { stripe.Key = "" })

			stripeProvider, _ := resolvePaymentProviders(
				config.StripeConfig{SecretKey: tc.secretKey, SuccessURL: "https://ryze.example/success"},
				config.PayPalConfig{},
				config.WebhookConfig{StripeWebhookSecret: tc.webhookKey},
			)

			methodMap := payments.NewMethodProviderMap(stripeProvider, stripeProvider, nil)
			advertised := methodMap.AvailableMethods()

			// MB WAY is served by the same Stripe provider, so both methods
			// appear and disappear together with the same configuration.
			advertisesCard := containsMethod(advertised, payments.PaymentMethodCard)
			advertisesMBWay := containsMethod(advertised, payments.PaymentMethodMBWay)

			if tc.wantEnabled {
				if !advertisesCard || !advertisesMBWay {
					t.Fatalf("expected card and mbway to be advertised when Stripe is fully configured, got %v", advertised)
				}
				if stripe.Key != tc.secretKey {
					t.Errorf("expected the Stripe key to be configured, got %q", stripe.Key)
				}
				return
			}

			if advertisesCard || advertisesMBWay {
				t.Fatalf("card and mbway must not be advertised when Stripe is not completely configured, got %v", advertised)
			}
			if stripeProvider != nil {
				t.Error("expected no Stripe provider when Stripe is not completely configured")
			}
			// A stale global key must never remain usable.
			if stripe.Key != "" {
				t.Errorf("expected the Stripe key to be cleared, got %q", stripe.Key)
			}
		})
	}
}

// TestResolvePaymentProviders_StripeFailureNeverEnablesPayPal verifies that
// providers are resolved independently: a misconfigured Stripe must not
// disturb a working PayPal configuration.
func TestResolvePaymentProviders_StripeFailureNeverEnablesPayPal(t *testing.T) {
	t.Cleanup(func() { stripe.Key = "" })

	stripeProvider, paypalProvider := resolvePaymentProviders(
		config.StripeConfig{SecretKey: "sk_test_123"},
		config.PayPalConfig{ClientID: "client-id", Secret: "secret", Mode: "sandbox", ReturnURL: "https://ryze.example/return", CancelURL: "https://ryze.example/cancel"},
		config.WebhookConfig{},
	)

	if stripeProvider != nil {
		t.Error("expected no Stripe provider without a webhook signing secret")
	}
	if paypalProvider == nil {
		t.Fatal("expected the PayPal provider to be configured independently of Stripe")
	}

	available := payments.NewMethodProviderMap(stripeProvider, stripeProvider, paypalProvider).AvailableMethods()
	if len(available) != 1 || available[0] != payments.PaymentMethodPayPal {
		t.Fatalf("expected only paypal to be advertised, got %v", available)
	}
}

// TestResolvePaymentProviders_PayPalOnly verifies the normal case where only
// PayPal is configured: no Stripe key is installed on the global client.
func TestResolvePaymentProviders_PayPalOnly(t *testing.T) {
	t.Cleanup(func() { stripe.Key = "" })

	stripeProvider, paypalProvider := resolvePaymentProviders(
		config.StripeConfig{},
		config.PayPalConfig{ClientID: "client-id", Secret: "secret", Mode: "sandbox"},
		config.WebhookConfig{},
	)

	if stripeProvider != nil {
		t.Error("expected no Stripe provider when Stripe is not configured")
	}
	if paypalProvider == nil {
		t.Error("expected the PayPal provider to be created")
	}
	if stripe.Key != "" {
		t.Errorf("expected no Stripe key to be installed, got %q", stripe.Key)
	}
}

// TestStripeEnabledGatesProviderAndWebhookEndpoint verifies the single
// complete-configuration rule: the card method and the Stripe webhook endpoint
// are gated by the same condition, so the two can never disagree.
func TestStripeEnabledGatesProviderAndWebhookEndpoint(t *testing.T) {
	cases := []struct {
		name       string
		secretKey  string
		webhookKey string
		want       bool
	}{
		{name: "fully configured", secretKey: "sk_test_123", webhookKey: "whsec_123", want: true},
		{name: "secret without webhook secret", secretKey: "sk_test_123", want: false},
		{name: "webhook secret without secret", webhookKey: "whsec_123", want: false},
		{name: "nothing configured", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripeEnabled(
				config.StripeConfig{SecretKey: tc.secretKey},
				config.WebhookConfig{StripeWebhookSecret: tc.webhookKey},
			)
			if got != tc.want {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

// TestPayPalEnabledIsTheSingleReadinessPredicate verifies that the PayPal
// provider and its webhook endpoint share one readiness rule: client id and
// secret present and a supported mode. An unsupported mode must disable
// PayPal instead of silently falling back to sandbox.
func TestPayPalEnabledIsTheSingleReadinessPredicate(t *testing.T) {
	cases := []struct {
		name     string
		clientID string
		secret   string
		mode     string
		want     bool
	}{
		{name: "fully configured sandbox", clientID: "client-id", secret: "secret", mode: "sandbox", want: true},
		{name: "fully configured live", clientID: "client-id", secret: "secret", mode: "live", want: true},
		{name: "mode with surrounding whitespace", clientID: "client-id", secret: "secret", mode: " Live ", want: true},
		{name: "unsupported mode disables PayPal", clientID: "client-id", secret: "secret", mode: "production", want: false},
		{name: "missing mode disables PayPal", clientID: "client-id", secret: "secret", mode: "", want: false},
		{name: "missing secret disables PayPal", clientID: "client-id", mode: "sandbox", want: false},
		{name: "missing client id disables PayPal", secret: "secret", mode: "sandbox", want: false},
		{name: "nothing configured", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := paypalEnabled(config.PayPalConfig{ClientID: tc.clientID, Secret: tc.secret, Mode: tc.mode})
			if got != tc.want {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

// TestPayPalEnabledMatchesProviderCreation verifies that every configuration
// the predicate accepts produces a provider and every configuration it rejects
// produces none, so the provider and the webhook gate can never disagree.
func TestPayPalEnabledMatchesProviderCreation(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.PayPalConfig
	}{
		{name: "sandbox", cfg: config.PayPalConfig{ClientID: "client-id", Secret: "secret", Mode: "sandbox"}},
		{name: "live", cfg: config.PayPalConfig{ClientID: "client-id", Secret: "secret", Mode: "live"}},
		{name: "unsupported mode", cfg: config.PayPalConfig{ClientID: "client-id", Secret: "secret", Mode: "production"}},
		{name: "missing secret", cfg: config.PayPalConfig{ClientID: "client-id", Mode: "sandbox"}},
		{name: "missing client id", cfg: config.PayPalConfig{Secret: "secret", Mode: "sandbox"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { stripe.Key = "" })

			_, paypalProvider := resolvePaymentProviders(config.StripeConfig{}, tc.cfg, config.WebhookConfig{})

			if paypalEnabled(tc.cfg) != (paypalProvider != nil) {
				t.Fatalf("predicate and provider creation disagree for %+v", tc.cfg)
			}
		})
	}
}

// TestCreatePayPalWebhookVerifierFailsClosedOnUnsupportedMode verifies the
// verifier never silently falls back to the sandbox API for a mode PayPal does
// not support: a mistyped mode must reject the verifier rather than mount an
// endpoint for a provider the rest of the stack has disabled.
func TestCreatePayPalWebhookVerifierFailsClosedOnUnsupportedMode(t *testing.T) {
	if _, err := createPayPalWebhookVerifier(config.PayPalConfig{ClientID: "client-id", Secret: "secret", Mode: "production"}); err == nil {
		t.Fatal("expected an unsupported mode to reject the webhook verifier")
	}

	client, err := createPayPalWebhookVerifier(config.PayPalConfig{ClientID: "client-id", Secret: "secret", Mode: "sandbox"})
	if err != nil {
		t.Fatalf("expected the sandbox mode to produce a verifier, got %v", err)
	}
	if client == nil {
		t.Fatal("expected a verifier client for the sandbox mode")
	}
}

func containsMethod(methods []payments.PaymentMethod, want payments.PaymentMethod) bool {
	for _, method := range methods {
		if method == want {
			return true
		}
	}
	return false
}
