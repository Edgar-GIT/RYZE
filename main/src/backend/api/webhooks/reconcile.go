package webhooks

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/models"
	"ryze/backend/services/purchases"
)

// reconcileLateSettlement handles a verified, contract-matched full payment for
// a purchase that is no longer pending (and not yet completed, which the caller
// acknowledges idempotently before reaching this helper). It is the backend's
// answer to a late settlement: a provider-confirmed payment must never be
// silently discarded, so a failed purchase is reconciled (failed -> completed
// plus entitlement) as long as the (user, program) pair has not resolved
// another way.
//
// Response semantics for a verified event:
//   - success -> 200, the purchase was reconciled atomically;
//   - reconciliation conflict or purchase moved on -> 200 with a diagnostic:
//     the payment cannot define a second sale and the failed record is retained
//     as the persisted, diagnosable recovery state, so no provider retry is
//     requested;
//   - entitlement integrity or transient failure -> 500 so the provider
//     retries the delivery.
//
// Every verdict is also recorded as a durable reconciliation event (see
// ReconciliationOutcomeRecorder) so a late settlement that could not grant a
// sale leaves a queryable record in addition to the failed purchase row. The
// record is best-effort: it never changes the HTTP response.
//
// This path can only be reached from a signature-verified provider event: the
// service accepts a purchase id with no authenticated owner, so it is never
// mounted on a client route.
//
// providerDB is the backend provider identifier stored on the durable record
// ("stripe" or "paypal"); logPrefix is the human-facing provider used in logs.
func reconcileLateSettlement(c *gin.Context, logPrefix, providerDB, eventID, purchaseID string, purchaseService purchases.Service, outcomes ReconciliationOutcomeRecorder) {
	ctx := c.Request.Context()
	result, err := purchaseService.ReconcilePurchase(ctx, purchaseID)
	if err != nil {
		switch {
		case errors.Is(err, purchases.ErrReconciliationConflict):
			log.Printf("[%s] late settlement for purchase %s (event %s) rejected: the pair already resolved, failed record retained as recovery state", logPrefix, purchaseID, eventID)
			recordOutcome(ctx, outcomes, providerDB, purchaseID, eventID, models.ReconciliationOutcomeConflictPairResolved)
			c.String(http.StatusOK, "reconciliation conflict")
		case errors.Is(err, purchases.ErrPurchaseNotCompleted):
			log.Printf("[%s] late settlement for purchase %s (event %s) observed the purchase moving on concurrently; outcome settled by the current state", logPrefix, purchaseID, eventID)
			recordOutcome(ctx, outcomes, providerDB, purchaseID, eventID, models.ReconciliationOutcomePurchaseMovedOn)
			c.String(http.StatusOK, "purchase moved on")
		case errors.Is(err, purchases.ErrEntitlementIntegrity):
			log.Printf("[%s] late settlement for purchase %s (event %s) hit an entitlement integrity violation", logPrefix, purchaseID, eventID)
			recordOutcome(ctx, outcomes, providerDB, purchaseID, eventID, models.ReconciliationOutcomeEntitlementIntegrity)
			c.String(http.StatusInternalServerError, "entitlement integrity")
		default:
			log.Printf("[%s] late settlement reconciliation failed for purchase %s (event %s): %v", logPrefix, purchaseID, eventID, err)
			recordOutcome(ctx, outcomes, providerDB, purchaseID, eventID, models.ReconciliationOutcomeTransientFailure)
			c.String(http.StatusInternalServerError, "reconciliation failed")
		}
		return
	}

	log.Printf("[%s] purchase %s reconciled to completed via a late verified settlement (event %s)", logPrefix, result.ID, eventID)
	recordOutcome(ctx, outcomes, providerDB, purchaseID, eventID, models.ReconciliationOutcomeReconciled)
	c.String(http.StatusOK, "reconciled")
}