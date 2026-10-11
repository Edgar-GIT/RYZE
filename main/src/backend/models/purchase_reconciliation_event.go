package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PurchaseReconciliationOutcome values describe the durable verdict a verified
// provider delivery left on a purchase. They are a fixed, documented vocabulary
// recorded by the trusted webhook layer so an operator can investigate an
// unresolved verified settlement without relying on unstructured logs. They are
// audit outcomes, not purchase statuses: the purchase status state machine
// (pending/completed/failed) is deliberately left unchanged.
const (
	// ReconciliationOutcomeCompleted records a provider-verified payment that
	// completed a pending purchase on its first delivery.
	ReconciliationOutcomeCompleted = "completed"
	// ReconciliationOutcomeAlreadyCompleted records a duplicate delivery of an
	// already-completed purchase, acknowledged idempotently.
	ReconciliationOutcomeAlreadyCompleted = "already_completed"
	// ReconciliationOutcomeReconciled records a late verified settlement that
	// atomically returned a failed purchase to completed.
	ReconciliationOutcomeReconciled = "reconciled"
	// ReconciliationOutcomeConflictPairResolved records a late settlement that
	// could not be granted because the (user, program) pair already resolved
	// through a completed purchase or an active entitlement.
	ReconciliationOutcomeConflictPairResolved = "conflict_pair_resolved"
	// ReconciliationOutcomePurchaseMovedOn records a reconcile observing the
	// purchase moving on concurrently; the outcome is settled by current state.
	ReconciliationOutcomePurchaseMovedOn = "purchase_moved_on"
	// ReconciliationOutcomeEntitlementIntegrity records a completed purchase
	// whose entitlement is missing — a data consistency violation requiring
	// investigation.
	ReconciliationOutcomeEntitlementIntegrity = "entitlement_integrity"
	// ReconciliationOutcomeTransientFailure records a transient failure the
	// provider will retry (5xx response). It is provisional: the retry's final
	// verdict overwrites it in the durable record.
	ReconciliationOutcomeTransientFailure = "transient_failure"
	// ReconciliationOutcomeAmountMismatch records a verified event whose amount
	// differs from the immutable purchase snapshot. It is permanent: retrying
	// cannot change it, so the delivery is acknowledged without a retry.
	ReconciliationOutcomeAmountMismatch = "amount_mismatch"
	// ReconciliationOutcomeCurrencyMismatch records a verified event whose
	// currency differs from the immutable purchase snapshot. It is permanent.
	ReconciliationOutcomeCurrencyMismatch = "currency_mismatch"
	// ReconciliationOutcomeMethodMismatch records a verified event whose
	// payment method cannot bind to the purchase's recorded method. It is
	// permanent.
	ReconciliationOutcomeMethodMismatch = "method_mismatch"
	// ReconciliationOutcomeReferenceMismatch records a verified event whose
	// purchase reference does not match the purchase it resolved to. It is
	// permanent.
	ReconciliationOutcomeReferenceMismatch = "reference_mismatch"
	// ReconciliationOutcomeProviderUncertain records a provider delivery that
	// does not yet report a settled payment (for example a still-unpaid
	// checkout session). It is provisional: a later settlement event records
	// its own final outcome.
	ReconciliationOutcomeProviderUncertain = "provider_uncertain"
	// ReconciliationOutcomeLateApproval records a verified late order approval
	// arriving after the purchase left the pending state. The order is never
	// captured, so no funds move.
	ReconciliationOutcomeLateApproval = "late_approval"
	// ReconciliationOutcomePermanentInvalid records a verified but malformed
	// delivery that can never complete a purchase and is not fixed by a retry.
	ReconciliationOutcomePermanentInvalid = "permanent_invalid"
	// ReconciliationOutcomeTestPurchaseEvent records a verified real-payment
	// event that resolved to a Test Mode purchase. It never completes anything
	// and marks the anomaly for investigation.
	ReconciliationOutcomeTestPurchaseEvent = "test_purchase_event"
)

// ProvisionalReconciliationOutcomes are the outcomes that describe an
// in-progress provider exchange rather than a settled verdict. A later
// delivery with the same (provider, event_id) overwrites them so the durable
// record ends with the final outcome instead of the intermediate one.
func ProvisionalReconciliationOutcomes() map[string]bool {
	return map[string]bool{
		ReconciliationOutcomeTransientFailure:   true,
		ReconciliationOutcomeProviderUncertain:  true,
	}
}

// OpenReconciliationOutcomes are the outcomes an operator should investigate:
// anything that is not a clean, resolved settlement (completed,
// already_completed or reconciled). The admin reconciliation view defaults to
// this set so it opens on what needs attention instead of burying it under
// routine settlements.
func OpenReconciliationOutcomes() map[string]bool {
	return map[string]bool{
		ReconciliationOutcomeConflictPairResolved:      true,
		ReconciliationOutcomePurchaseMovedOn:           true,
		ReconciliationOutcomeEntitlementIntegrity:      true,
		ReconciliationOutcomeTransientFailure:          true,
		ReconciliationOutcomeAmountMismatch:            true,
		ReconciliationOutcomeCurrencyMismatch:          true,
		ReconciliationOutcomeMethodMismatch:            true,
		ReconciliationOutcomeReferenceMismatch:         true,
		ReconciliationOutcomeProviderUncertain:         true,
		ReconciliationOutcomeLateApproval:              true,
		ReconciliationOutcomePermanentInvalid:          true,
		ReconciliationOutcomeTestPurchaseEvent:         true,
	}
}

// PurchasedReconciliationOutcomeCarriers are the clean settlement outcomes.
func ResolvedReconciliationOutcomes() map[string]bool {
	return map[string]bool{
		ReconciliationOutcomeCompleted:        true,
		ReconciliationOutcomeAlreadyCompleted: true,
		ReconciliationOutcomeReconciled:       true,
	}
}

// IsValidReconciliationOutcome reports whether an outcome belongs to the fixed
// documented vocabulary. The repository and admin filter reject unknown codes
// so the audit table never accumulates open-ended free text.
func IsValidReconciliationOutcome(outcome string) bool {
	switch outcome {
	case ReconciliationOutcomeCompleted,
		ReconciliationOutcomeAlreadyCompleted,
		ReconciliationOutcomeReconciled,
		ReconciliationOutcomeConflictPairResolved,
		ReconciliationOutcomePurchaseMovedOn,
		ReconciliationOutcomeEntitlementIntegrity,
		ReconciliationOutcomeTransientFailure,
		ReconciliationOutcomeAmountMismatch,
		ReconciliationOutcomeCurrencyMismatch,
		ReconciliationOutcomeMethodMismatch,
		ReconciliationOutcomeReferenceMismatch,
		ReconciliationOutcomeProviderUncertain,
		ReconciliationOutcomeLateApproval,
		ReconciliationOutcomePermanentInvalid,
		ReconciliationOutcomeTestPurchaseEvent:
		return true
	default:
		return false
	}
}

// PurchaseReconciliationEvent is the durable, minimum-safe diagnostic record
// of what a verified provider delivery did to a purchase. It is written only
// by the trusted webhook layer and read only by the restricted ADMIN_2
// reconciliation surface. It carries no secrets, signatures, provider payloads,
// questionnaire answers or nutrition data: just the purchase, the provider, the
// provider event id, the outcome code and the first-seen timestamp.
//
// The row is append-only and deliberately never soft-deleted: audit records
// must survive exactly as written, so the soft-delete convention does not apply
// here. One row exists per (provider, event_id) delivery; a retry upserts the
// provisional outcome with the final verdict instead of writing a second row.
type PurchaseReconciliationEvent struct {
	ID         string    `gorm:"column:id;type:char(36);primaryKey" json:"id"`
	PurchaseID string    `gorm:"column:purchase_id;type:char(36);not null" json:"purchase_id"`
	Provider   string    `gorm:"column:provider;type:varchar(16);not null" json:"provider"`
	EventID    string    `gorm:"column:event_id;type:varchar(128);not null" json:"event_id"`
	Outcome    string    `gorm:"column:outcome;type:varchar(32);not null" json:"outcome"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	// Purchase is the resolved purchase for read models. It is preloaded by the
	// administrative query; the webhook recorder never reads it back.
	Purchase *Purchase `gorm:"foreignKey:PurchaseID;references:ID" json:"-"`
}

func (p *PurchaseReconciliationEvent) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}