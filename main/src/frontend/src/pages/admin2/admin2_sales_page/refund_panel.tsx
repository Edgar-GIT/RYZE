import { RotateCcw } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/button/button";
import { refundAdminPurchase, type AdminPurchase, type RefundReason } from "@/services/admin2_api";
import { ApiError } from "@utils/http_client";

import styles from "./refund_panel.module.css";

const REASONS: Array<{ id: RefundReason; label: string; hint: string }> = [
  { id: "non_conformity", label: "Content not as described", hint: "The plan does not match its description or does not work (conformity of digital content)." },
  { id: "withdrawal", label: "Right of withdrawal", hint: "The client withdrew where the law allows it (for example a Premium Level 2 not yet started)." },
  { id: "duplicate", label: "Duplicate payment", hint: "The client paid twice for the same thing." },
  { id: "goodwill", label: "Goodwill", hint: "A commercial gesture." }
];

const PROVIDER_LABEL: Record<string, string> = { card: "Stripe (card)", mbway: "Stripe (MB WAY)", paypal: "PayPal" };

// RefundPanel refunds a completed purchase (owner request, 2026-11-24). The
// access the purchase gave ends at once — including a Premium Level 2 personal
// plan — and the sale leaves the trainer's earnings.
export const RefundPanel = ({ purchase, onRefunded }: { purchase: AdminPurchase; onRefunded: (purchase: AdminPurchase) => void }) => {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState<RefundReason>("non_conformity");
  const [note, setNote] = useState("");
  const [alreadyRefunded, setAlreadyRefunded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const method = purchase.payment_method ?? "";
  const paid = !purchase.test && purchase.price_minor_units > 0;
  const automatic = paid && (method === "card" || method === "mbway");

  const refund = async () => {
    const money = !paid
      ? "No money was paid (test or free purchase)."
      : alreadyRefunded
        ? "You confirm the money was already returned in the provider's dashboard."
        : automatic
          ? "The money goes back to the client through Stripe."
          : "";
    if (!window.confirm(`Refund this purchase? ${money} The client loses access to the plan right away and the sale leaves the trainer's earnings.`)) {
      return;
    }
    setBusy(true);
    setError("");
    try {
      const result = await refundAdminPurchase(purchase.id, { reason, note: note.trim(), alreadyRefunded });
      onRefunded({ ...purchase, status: result.status, access: false, refunded_at: result.refunded_at, refund_reason: result.refund_reason });
    } catch (caught) {
      const code = caught instanceof ApiError ? caught.code : "";
      setError(
        code === "REFUND_MANUAL_REQUIRED"
          ? "RYZE cannot return this payment automatically. Refund it in the provider's dashboard, then tick “Already refunded” and confirm."
          : code === "REFUND_PROVIDER_FAILED"
            ? "The provider could not refund the payment. Try again, or refund it in its dashboard and record it here."
            : code === "PURCHASE_NOT_REFUNDABLE"
              ? "Only a completed purchase can be refunded."
              : "The refund could not be completed. Try again."
      );
    } finally {
      setBusy(false);
    }
  };

  if (purchase.status === "refunded") {
    return (
      <p className={styles.done}>
        <RotateCcw size={15} aria-hidden="true" /> Refunded
        {purchase.refund_reason ? ` · ${REASONS.find((item) => item.id === purchase.refund_reason)?.label ?? purchase.refund_reason}` : ""}
      </p>
    );
  }
  if (purchase.status !== "completed") return null;
  if (!open) {
    return (
      <Button variant="secondary" size="small" icon={<RotateCcw size={14} />} iconPosition="left" onClick={() => setOpen(true)}>
        Refund…
      </Button>
    );
  }
  return (
    <section className={styles.panel} aria-label="Refund">
      <h3>Refund this purchase</h3>
      <fieldset className={styles.reasons}>
        <legend>Reason</legend>
        {REASONS.map((item) => (
          <label key={item.id} className={styles.reason}>
            <input type="radio" name="refund-reason" checked={reason === item.id} onChange={() => setReason(item.id)} />
            <span>
              <strong>{item.label}</strong>
              <small>{item.hint}</small>
            </span>
          </label>
        ))}
      </fieldset>
      <label className={styles.field}>
        <span>Note (internal, optional)</span>
        <textarea value={note} maxLength={500} rows={2} onChange={(event) => setNote(event.target.value)} />
      </label>
      {paid ? (
        <>
          <p className={styles.hint}>
            Paid with {PROVIDER_LABEL[method] ?? "an unknown method"}.{" "}
            {automatic
              ? "RYZE returns the money through Stripe."
              : "Refund the payment in the provider's dashboard first, then record it here."}
          </p>
          <label className={styles.check}>
            <input type="checkbox" checked={alreadyRefunded} onChange={(event) => setAlreadyRefunded(event.target.checked)} />
            <span>Already refunded in the provider's dashboard (only record it here)</span>
          </label>
        </>
      ) : (
        <p className={styles.hint}>No money was paid (test or free purchase): only the access ends.</p>
      )}
      {error ? <p className={styles.error} role="alert">{error}</p> : null}
      <div className={styles.actions}>
        <Button variant="secondary" size="small" disabled={busy} onClick={() => setOpen(false)}>
          Cancel
        </Button>
        <Button variant="primary" size="small" disabled={busy || (paid && !automatic && !alreadyRefunded)} onClick={() => void refund()}>
          {busy ? "Refunding…" : "Refund and remove access"}
        </Button>
      </div>
    </section>
  );
};
