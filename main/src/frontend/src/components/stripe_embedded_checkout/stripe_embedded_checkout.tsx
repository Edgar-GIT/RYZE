import { Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import styles from "./stripe_embedded_checkout.module.css";

// Stripe.js is always loaded from Stripe (never bundled), as Stripe requires.
const STRIPE_JS = "https://js.stripe.com/endive/stripe.js";

interface EmbeddedCheckoutPage {
  mount: (target: HTMLElement | string) => void;
  unmount?: () => void;
  destroy?: () => void;
}

interface StripeInstance {
  createEmbeddedCheckoutPage: (options: { fetchClientSecret: () => Promise<string> }) => Promise<EmbeddedCheckoutPage>;
}

declare global {
  interface Window {
    Stripe?: (publishableKey: string) => StripeInstance;
  }
}

let loading: Promise<void> | null = null;

const loadStripeJs = (): Promise<void> => {
  if (window.Stripe) return Promise.resolve();
  if (!loading) {
    loading = new Promise((resolve, reject) => {
      const script = document.createElement("script");
      script.src = STRIPE_JS;
      script.async = true;
      script.onload = () => resolve();
      script.onerror = () => {
        loading = null;
        reject(new Error("Stripe.js could not be loaded"));
      };
      document.head.appendChild(script);
    });
  }
  return loading;
};

interface StripeEmbeddedCheckoutProps {
  publishableKey: string;
  clientSecret: string;
  loadingLabel: string;
  errorLabel: string;
}

// StripeEmbeddedCheckout mounts Stripe's own Checkout form (card number,
// expiry, CVC with the brand recognised from the number; or MB WAY) inside
// the RYZE checkout (owner request, 2026-11-23: always use the provider's
// screens). Card data goes straight to Stripe; when the payment completes,
// Stripe sends the buyer to the return URL, where RYZE verifies it.
export const StripeEmbeddedCheckout = ({ publishableKey, clientSecret, loadingLabel, errorLabel }: StripeEmbeddedCheckoutProps) => {
  const target = useRef<HTMLDivElement>(null);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");

  useEffect(() => {
    let cancelled = false;
    let page: EmbeddedCheckoutPage | null = null;
    loadStripeJs()
      .then(async () => {
        if (cancelled || !window.Stripe || !target.current) return;
        page = await window.Stripe(publishableKey).createEmbeddedCheckoutPage({ fetchClientSecret: () => Promise.resolve(clientSecret) });
        if (cancelled) {
          page.destroy?.();
          return;
        }
        page.mount(target.current);
        setState("ready");
      })
      .catch(() => {
        if (!cancelled) setState("error");
      });
    return () => {
      cancelled = true;
      page?.destroy?.();
    };
  }, [clientSecret, publishableKey]);

  return (
    <div className={styles.frame}>
      {state === "loading" ? (
        <p className={styles.status}>
          <Loader2 className={styles.spin} aria-hidden="true" /> {loadingLabel}
        </p>
      ) : null}
      {state === "error" ? (
        <p className={styles.error} role="alert">
          {errorLabel}
        </p>
      ) : null}
      <div ref={target} className={styles.mount} />
    </div>
  );
};
