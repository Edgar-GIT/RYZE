import { Ban, Copy, Plus, TicketPercent } from "lucide-react";
import { type FormEvent, useCallback, useEffect, useMemo, useState } from "react";

import { Button } from "@/components/button/button";
import {
  createDiscountCode,
  disableDiscountCode,
  fetchDiscountCodes,
  type DiscountCode,
  type DiscountScope
} from "@/services/discount_codes_api";
import { ApiError } from "@utils/http_client";
import { joinClassNames } from "@utils/class_names";

import styles from "./discount_codes_manager.module.css";

const SCOPES: Array<{ id: DiscountScope; label: string }> = [
  { id: "all", label: "Every paid plan" },
  { id: "premium_level_1", label: "Premium Level 1" },
  { id: "premium_level_2", label: "Premium Level 2" },
  { id: "trainer_plans", label: "Trainers' plans" }
];
const DAY_PRESETS = [1, 3, 7, 14, 30];
const HOUR_PRESETS = [1, 2, 6, 12, 24];
const PERCENT_PRESETS = [10, 15, 20, 25, 50];

const dateTime = new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });

const statusOf = (code: DiscountCode, now: number): { label: string; tone: string } => {
  if (code.disabled_at) return { label: "Disabled", tone: styles.off };
  if (Date.parse(code.expires_at) <= now) return { label: "Expired", tone: styles.off };
  if (code.max_uses !== null && code.uses >= code.max_uses) return { label: "Used up", tone: styles.off };
  return { label: "Active", tone: styles.on };
};

// DiscountCodesManager creates, lists and disables discount codes (owner
// request, 2026-11-24). Shared by both administrator consoles. A code is
// usable for at most 30 days, chosen in days or, for 24 hours or less, hours.
export const DiscountCodesManager = () => {
  const [codes, setCodes] = useState<DiscountCode[] | null>(null);
  const [loadError, setLoadError] = useState(false);
  const [code, setCode] = useState("");
  const [percent, setPercent] = useState("20");
  const [scope, setScope] = useState<DiscountScope>("all");
  const [unit, setUnit] = useState<"days" | "hours">("days");
  const [amount, setAmount] = useState("7");
  const [maxUses, setMaxUses] = useState("");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null);
  const [now, setNow] = useState(() => Date.now());

  const load = useCallback(async () => {
    setLoadError(false);
    try {
      setCodes(await fetchDiscountCodes());
      setNow(Date.now());
    } catch {
      setLoadError(true);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const limit = unit === "days" ? 30 : 24;
  const amountNumber = Number(amount);
  const durationHours = unit === "days" ? amountNumber * 24 : amountNumber;
  const durationValid = Number.isInteger(amountNumber) && amountNumber >= 1 && amountNumber <= limit;
  const until = useMemo(
    () => (durationValid ? dateTime.format(new Date(Date.now() + durationHours * 3_600_000)) : ""),
    [durationHours, durationValid]
  );

  const create = async (event: FormEvent) => {
    event.preventDefault();
    const percentNumber = Number(percent);
    const uses = maxUses.trim() === "" ? null : Number(maxUses);
    if (!/^[A-Z0-9][A-Z0-9_-]{2,31}$/.test(code)) {
      setMessage({ ok: false, text: "The code has 3 to 32 capital letters, digits, - or _." });
      return;
    }
    if (!Number.isInteger(percentNumber) || percentNumber < 1 || percentNumber > 90) {
      setMessage({ ok: false, text: "The discount is 1% to 90%." });
      return;
    }
    if (!durationValid) {
      setMessage({ ok: false, text: unit === "days" ? "Choose 1 to 30 days." : "Choose 1 to 24 hours." });
      return;
    }
    if (uses !== null && (!Number.isInteger(uses) || uses < 1)) {
      setMessage({ ok: false, text: "Leave the uses empty for no limit, or enter a whole number." });
      return;
    }
    setSaving(true);
    setMessage(null);
    try {
      const created = await createDiscountCode({ code, percent_off: percentNumber, applies_to: scope, duration_hours: durationHours, max_uses: uses });
      setMessage({ ok: true, text: `${created.code} created: ${created.percent_off}% off until ${dateTime.format(new Date(created.expires_at))}.` });
      setCode("");
      await load();
    } catch (error) {
      setMessage({
        ok: false,
        text:
          error instanceof ApiError && error.code === "DISCOUNT_CODE_TAKEN"
            ? "This code already exists. Choose another one."
            : "The code could not be created. Check the details and try again."
      });
    } finally {
      setSaving(false);
    }
  };

  const disable = async (target: DiscountCode) => {
    if (!window.confirm(`Disable ${target.code}? Nobody can use it any more; purchases already made keep their price.`)) return;
    try {
      await disableDiscountCode(target.id);
      await load();
    } catch {
      setMessage({ ok: false, text: "The code could not be disabled. Try again." });
    }
  };

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <span className={styles.icon} aria-hidden="true">
          <TicketPercent />
        </span>
        <div>
          <h1>Discount codes</h1>
          <p>A percentage off paid plans, usable for up to 30 days. Each person can use a code once; buyers enter it on the payment screen.</p>
        </div>
      </header>

      <form className={styles.card} onSubmit={(event) => void create(event)}>
        <h2>New code</h2>
        <div className={styles.grid}>
          <label className={styles.field}>
            <span>Code</span>
            <input
              value={code}
              maxLength={32}
              placeholder="LAUNCH20"
              autoCapitalize="characters"
              spellCheck={false}
              onChange={(event) => setCode(event.target.value.toUpperCase().replace(/\s/g, ""))}
              required
            />
          </label>
          <label className={styles.field}>
            <span>Plans</span>
            <select value={scope} onChange={(event) => setScope(event.target.value as DiscountScope)}>
              {SCOPES.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.label}
                </option>
              ))}
            </select>
          </label>
        </div>

        <div className={styles.field}>
          <span>Discount</span>
          <div className={styles.inline}>
            <input className={styles.short} type="number" inputMode="numeric" min={1} max={90} value={percent} onChange={(event) => setPercent(event.target.value)} aria-label="Percentage off" />
            <span className={styles.unit}>%</span>
            <div className={styles.chips} role="group" aria-label="Common discounts">
              {PERCENT_PRESETS.map((value) => (
                <button key={value} type="button" aria-pressed={percent === String(value)} onClick={() => setPercent(String(value))}>
                  {value}%
                </button>
              ))}
            </div>
          </div>
        </div>

        <div className={styles.field}>
          <span>Valid for</span>
          <div className={styles.segmented} role="group" aria-label="Unit">
            <button type="button" aria-pressed={unit === "days"} onClick={() => { setUnit("days"); setAmount("7"); }}>
              Days
            </button>
            <button type="button" aria-pressed={unit === "hours"} onClick={() => { setUnit("hours"); setAmount("24"); }}>
              Hours (24h or less)
            </button>
          </div>
          <div className={styles.inline}>
            <input
              className={styles.short}
              type="number"
              inputMode="numeric"
              min={1}
              max={limit}
              value={amount}
              onChange={(event) => setAmount(event.target.value)}
              aria-label={unit === "days" ? "Number of days" : "Number of hours"}
            />
            <span className={styles.unit}>{unit === "days" ? "days" : "hours"}</span>
            <div className={styles.chips} role="group" aria-label="Quick choices">
              {(unit === "days" ? DAY_PRESETS : HOUR_PRESETS).map((value) => (
                <button key={value} type="button" aria-pressed={amount === String(value)} onClick={() => setAmount(String(value))}>
                  {value}
                  {unit === "days" ? (value === 1 ? " day" : " days") : "h"}
                </button>
              ))}
            </div>
          </div>
          <small className={styles.hint}>{durationValid ? `Usable until ${until}.` : `Choose 1 to ${limit} ${unit}.`}</small>
        </div>

        <label className={styles.field}>
          <span>Maximum uses (optional)</span>
          <input className={styles.short} type="number" inputMode="numeric" min={1} placeholder="No limit" value={maxUses} onChange={(event) => setMaxUses(event.target.value)} />
        </label>

        {message ? (
          <p className={message.ok ? styles.ok : styles.error} role={message.ok ? "status" : "alert"}>
            {message.text}
          </p>
        ) : null}
        <div>
          <Button type="submit" variant="primary" size="small" icon={<Plus size={15} />} iconPosition="left" disabled={saving}>
            {saving ? "Creating…" : "Create code"}
          </Button>
        </div>
      </form>

      <section className={styles.card} aria-label="Codes">
        <h2>Codes</h2>
        {loadError ? (
          <p className={styles.error}>The codes could not be loaded.</p>
        ) : codes === null ? (
          <p className={styles.hint}>Loading…</p>
        ) : codes.length === 0 ? (
          <p className={styles.hint}>No codes yet.</p>
        ) : (
          <ul className={styles.list}>
            {codes.map((item) => {
              const status = statusOf(item, now);
              const active = status.tone === styles.on;
              return (
                <li key={item.id} className={styles.row}>
                  <div className={styles.rowMain}>
                    <strong className={styles.codeText}>{item.code}</strong>
                    <span className={joinClassNames(styles.badge, status.tone)}>{status.label}</span>
                  </div>
                  <span className={styles.meta}>
                    {item.percent_off}% · {SCOPES.find((scopeItem) => scopeItem.id === item.applies_to)?.label ?? item.applies_to} · until{" "}
                    {dateTime.format(new Date(item.expires_at))} · {item.uses}
                    {item.max_uses !== null ? ` / ${item.max_uses}` : ""} uses
                  </span>
                  <div className={styles.rowActions}>
                    <button type="button" onClick={() => void navigator.clipboard?.writeText(item.code)} aria-label={`Copy ${item.code}`}>
                      <Copy aria-hidden="true" />
                    </button>
                    {active ? (
                      <button type="button" onClick={() => void disable(item)} aria-label={`Disable ${item.code}`}>
                        <Ban aria-hidden="true" />
                      </button>
                    ) : null}
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </section>
    </div>
  );
};
