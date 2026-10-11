package webhooks

import (
	"context"
	"log"
)

// ReconciliationOutcomeRecorder durably records what a verified provider
// delivery did to a purchase. It is implemented by the reconciliation event
// repository (wired in the routes layer) so the webhook verdicts that cannot
// be recovered from the purchase row alone — conflicts, mismatches, integrity
// violations, provisional retries — survive as queryable records instead of
// unstructured log lines.
//
// Recording is best-effort and never changes the webhook response: a failure
// to persist the diagnostic is logged and the fulfillment verdict (the HTTP
// code the provider acts on) is decided strictly by the purchase service. An
// audit outage must never make a confirmed payment look failed, nor an
// unconfirmed one look settled.
type ReconciliationOutcomeRecorder interface {
	Record(ctx context.Context, purchaseID, provider, eventID, outcome string) error
}

// recordOutcome writes one durable reconciliation outcome for a delivery,
// ignoring failures (see ReconciliationOutcomeRecorder). The provider value
// stored on the record is the backend provider identifier ("stripe" or
// "paypal"), while the log prefix keeps the human-facing provider name.
// A missing purchase id or event id means there is nothing safe to attach the
// record to, so nothing is written (those cases remain in the webhook logs).
func recordOutcome(ctx context.Context, rec ReconciliationOutcomeRecorder, provider, purchaseID, eventID, outcome string) {
	if rec == nil || purchaseID == "" || eventID == "" || outcome == "" {
		return
	}
	if err := rec.Record(ctx, purchaseID, provider, eventID, outcome); err != nil {
		log.Printf("[%s] failed to record reconciliation outcome %q for purchase %s (event %s): %v", provider, outcome, purchaseID, eventID, err)
	}
}