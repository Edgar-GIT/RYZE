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

func containsMethod(methods []payments.PaymentMethod, want payments.PaymentMethod) bool {
	for _, method := range methods {
		if method == want {
			return true
		}
	}
	return false
}
