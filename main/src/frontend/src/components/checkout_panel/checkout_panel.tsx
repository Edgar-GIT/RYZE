import { CheckCircle2, CreditCard, FileText, FlaskConical, Loader2, Lock, RefreshCw, ShieldCheck, Smartphone, Wallet } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
import { Link } from "react-router-dom";

import { CheckoutTaxIdField } from "@/components/checkout_tax_id/checkout_tax_id_field";
import { isValidOptionalTaxId } from "@/components/checkout_tax_id/tax_id";
import { useTestMode } from "@/components/test_mode/test_mode_context";
import { isValidTestEmail } from "@/components/test_mode/test_mode_email";
import { TestModeEmailField } from "@/components/test_mode/test_mode_email_field";
import { useProgramCheckout } from "@/hooks/use_program_checkout";
import { rememberAfterLogin } from "@/services/after_login";
import { receiptPath } from "@/services/invoices_api";
import { type Copy, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./checkout_panel.module.css";

interface CheckoutPanelProps {
  programId: string;
  // What is being bought, as the order summary shows it.
  itemTitle: string;
  itemCategory: string;
  priceMinor: number;
  // Lines under the total (what the purchase includes).
  includes: string[];
  // Where to go once the purchase is confirmed.
  done: { to: string; label: string };
  // Extra content shown with the confirmation (e.g. what happens next).
  successNote?: ReactNode;
  loginReturn: string;
  // Optional: run once when the purchase is confirmed (for an immediate
  // hand-off to the product's post-purchase experience).
  onSuccess?: () => void;
}

const COPY: Copy<{
  title: string;
  order: string;
  total: string;
  oneTime: string;
  method: string;
  pay: (price: string) => string;
  payTest: string;
  testMode: string;
  unavailable: string;
  unavailableText: string;
  secure: string;
  checking: string;
  working: string;
  redirecting: string;
  confirming: string;
  guest: string;
  guestText: string;
  login: string;
  owned: string;
  ownedText: string;
  success: string;
  successText: string;
  cancelled: string;
  cancelledText: string;
  resume: string;
  retry: string;
  receipt: string;
  errors: Record<string, string>;
}> = {
  en: {
    title: "Payment",
    order: "Your order",
    total: "Total",
    oneTime: "One-time payment · no subscription",
    method: "Payment method",
    pay: (price) => `Pay ${price}`,
    payTest: "Complete Test Mode purchase",
    testMode: "Test Mode is active: the purchase completes without any payment provider.",
    unavailable: "Payments are not available yet",
    unavailableText: "No payment provider is configured in this environment, so no payment can be started. Nothing was charged.",
    secure: "The payment is completed on the provider's secure page; RYZE never sees your card.",
    checking: "Checking your access…",
    working: "Preparing your payment…",
    redirecting: "Opening the secure payment page…",
    confirming: "Confirming your payment…",
    guest: "Sign in to pay",
    guestText: "Your purchase is tied to your RYZE account.",
    login: "Sign in",
    owned: "You already own this",
    ownedText: "There is nothing to pay.",
    success: "Payment confirmed",
    successText: "Your plan is now in your account.",
    cancelled: "Payment cancelled",
    cancelledText: "No money was taken. You can try again whenever you want.",
    resume: "Resume payment",
    retry: "Try again",
    receipt: "View purchase receipt",
    errors: {}
  },
  pt: {
    title: "Pagamento",
    order: "A tua encomenda",
    total: "Total",
    oneTime: "Pagamento único · sem subscrição",
    method: "Método de pagamento",
    pay: (price) => `Pagar ${price}`,
    payTest: "Concluir compra em Modo de Teste",
    testMode: "O Modo de Teste está ativo: a compra conclui-se sem nenhum fornecedor de pagamento.",
    unavailable: "Os pagamentos ainda não estão disponíveis",
    unavailableText: "Não há nenhum fornecedor de pagamento configurado neste ambiente, por isso não é possível iniciar um pagamento. Nada foi cobrado.",
    secure: "O pagamento é feito na página segura do fornecedor; a RYZE nunca vê o teu cartão.",
    checking: "A verificar o teu acesso…",
    working: "A preparar o pagamento…",
    redirecting: "A abrir a página de pagamento segura…",
    confirming: "A confirmar o pagamento…",
    guest: "Inicia sessão para pagar",
    guestText: "A compra fica associada à tua conta RYZE.",
    login: "Iniciar sessão",
    owned: "Já tens este plano",
    ownedText: "Não há nada a pagar.",
    success: "Pagamento confirmado",
    successText: "O teu plano já está na tua conta.",
    cancelled: "Pagamento cancelado",
    cancelledText: "Não foi cobrado nenhum valor. Podes tentar outra vez quando quiseres.",
    resume: "Retomar pagamento",
    retry: "Tentar de novo",
    receipt: "Ver comprovativo de compra",
    errors: {
      "Enter a valid NIF or leave it empty.": "Introduz um NIF válido ou deixa o campo vazio.",
      "We could not start your purchase. No money was taken at this point.": "Não foi possível iniciar a compra. Não foi cobrado nenhum valor.",
      "We could not start the checkout. No money was taken at this point.": "Não foi possível iniciar o pagamento. Não foi cobrado nenhum valor.",
      "We could not confirm your payment.": "Não foi possível confirmar o pagamento.",
      "We could not complete this Test Mode purchase. Please try again.": "Não foi possível concluir a compra em Modo de Teste. Tenta de novo.",
      "Enter a valid test email address.": "Introduz um email de teste válido.",
      "We could not verify your access. Please try again.": "Não foi possível verificar o teu acesso. Tenta de novo.",
      "Checkout is temporarily unavailable. Please try again later.": "O pagamento está temporariamente indisponível. Tenta mais tarde.",
      "Checkout is temporarily unavailable. Please try again.": "O pagamento está temporariamente indisponível. Tenta de novo."
    }
  }
};

const METHOD_ICON: Record<string, typeof CreditCard> = { card: CreditCard, mbway: Smartphone, paypal: Wallet };

// CheckoutPanel is the payment step of every paid RYZE journey: the order
// summary with the fixed price, the payment methods the backend advertises
// (never hard-coded), the Test Mode shortcut and honest states when nothing
// can be charged. The purchase lifecycle is the shared useProgramCheckout
// hook; the backend stays authoritative for every outcome.
export const CheckoutPanel = ({ programId, itemTitle, itemCategory, priceMinor, includes, done, successNote, loginReturn, onSuccess }: CheckoutPanelProps) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const { isActive: testModeActive } = useTestMode();
  const checkout = useProgramCheckout({ programId, free: false, testModeActive, enabled: Boolean(programId) });
  const price = new Intl.NumberFormat(locale === "pt" ? "pt-PT" : "en-IE", { style: "currency", currency: "EUR" }).format(priceMinor / 100);
  const state = checkout.state;
  const busy = ["creating", "negotiating", "redirecting", "processing"].includes(state.status);
  const taxIdValid = isValidOptionalTaxId(checkout.taxId);
  const canPay = taxIdValid && (testModeActive ? isValidTestEmail(checkout.testEmail) : checkout.paymentMethods.length > 0);
  const translate = (message: string) => text.errors[message] ?? message;
  const successHandled = useRef(false);

  useEffect(() => {
    if (state.status === "success" && !successHandled.current) {
      successHandled.current = true;
      onSuccess?.();
    }
  }, [onSuccess, state.status]);

  if (state.status === "success" || state.status === "owned") {
    const success = state.status === "success";
    return (
      <section className={joinClassNames(styles.panel, styles.done)} aria-live="polite">
        <span className={styles.doneIcon} aria-hidden="true">
          <CheckCircle2 />
        </span>
        <h2 className={styles.doneTitle}>{success ? text.success : text.owned}</h2>
        <p className={styles.doneText}>{success ? text.successText : text.ownedText}</p>
        {successNote}
        <Link className={styles.primary} to={done.to}>
          {done.label}
        </Link>
        {success ? (
          <Link className={styles.secondary} to={receiptPath(state.purchase.id)}>
            <FileText aria-hidden="true" /> {text.receipt}
          </Link>
        ) : null}
      </section>
    );
  }

  return (
    <section className={styles.panel} aria-labelledby="checkout-title">
      <h2 id="checkout-title" className={styles.title}>
        <Lock aria-hidden="true" /> {text.title}
      </h2>

      <div className={styles.order}>
        <p className={styles.orderLabel}>{text.order}</p>
        <div className={styles.orderItem}>
          <span>
            <strong>{itemTitle}</strong>
            <em>{itemCategory}</em>
          </span>
          <b>{price}</b>
        </div>
        {includes.length > 0 ? (
          <ul className={styles.includes}>
            {includes.map((line) => (
              <li key={line}>
                <CheckCircle2 aria-hidden="true" />
                {line}
              </li>
            ))}
          </ul>
        ) : null}
        <div className={styles.total}>
          <span>{text.total}</span>
          <strong>{price}</strong>
        </div>
        <p className={styles.oneTime}>{text.oneTime}</p>
      </div>

      {state.status === "checking" ? (
        <p className={styles.status}>
          <Loader2 className={styles.spin} aria-hidden="true" /> {text.checking}
        </p>
      ) : state.status === "guest" ? (
        <div className={styles.notice}>
          <strong>{text.guest}</strong>
          <p>{text.guestText}</p>
          <Link className={styles.primary} to="/login" onClick={() => rememberAfterLogin(loginReturn)}>
            {text.login}
          </Link>
        </div>
      ) : state.status === "cancelled" ? (
        <div className={styles.notice}>
          <strong>{text.cancelled}</strong>
          <p>{text.cancelledText}</p>
          <button type="button" className={styles.primary} onClick={() => void checkout.retryPurchase(state.purchaseId)}>
            {text.resume}
          </button>
        </div>
      ) : state.status === "error" ? (
        <div className={joinClassNames(styles.notice, styles.noticeError)} role="alert">
          <p>{translate(state.message)}</p>
          {testModeActive ? (
            <TestModeEmailField value={checkout.testEmail} onChange={checkout.setTestEmail} />
          ) : null}
          <CheckoutTaxIdField value={checkout.taxId} onChange={checkout.setTaxId} />
          <button
            type="button"
            className={styles.secondary}
            disabled={!taxIdValid || (testModeActive && !isValidTestEmail(checkout.testEmail))}
            onClick={() => void checkout.buy()}
          >
            <RefreshCw aria-hidden="true" /> {text.retry}
          </button>
        </div>
      ) : (
        <>
          {testModeActive ? (
            <>
              <p className={styles.testMode}>
                <FlaskConical aria-hidden="true" /> {text.testMode}
              </p>
              <TestModeEmailField value={checkout.testEmail} onChange={checkout.setTestEmail} />
            </>
          ) : checkout.paymentMethods.length > 0 ? (
            <fieldset className={styles.methods}>
              <legend>{text.method}</legend>
              {checkout.paymentMethods.map((method) => {
                const Icon = METHOD_ICON[method.method] ?? CreditCard;
                return (
                  <label
                    key={method.method}
                    className={joinClassNames(styles.method, checkout.selectedMethod === method.method && styles.methodActive)}
                  >
                    <input
                      type="radio"
                      name="payment-method"
                      value={method.method}
                      checked={checkout.selectedMethod === method.method}
                      onChange={() => checkout.selectMethod(method.method)}
                    />
                    <Icon aria-hidden="true" />
                    <span>{method.label}</span>
                  </label>
                );
              })}
            </fieldset>
          ) : (
            <div className={styles.notice}>
              <strong>{text.unavailable}</strong>
              <p>{text.unavailableText}</p>
            </div>
          )}

          {testModeActive || checkout.paymentMethods.length > 0 ? (
            <CheckoutTaxIdField value={checkout.taxId} onChange={checkout.setTaxId} disabled={busy} />
          ) : null}

          <button type="button" className={styles.primary} disabled={!canPay || busy} onClick={() => void checkout.buy()}>
            {busy ? <Loader2 className={styles.spin} aria-hidden="true" /> : <Lock aria-hidden="true" />}
            {state.status === "redirecting"
              ? text.redirecting
              : state.status === "processing"
                ? text.confirming
                : busy
                  ? text.working
                  : testModeActive
                    ? text.payTest
                    : text.pay(price)}
          </button>
          <p className={styles.secure}>
            <ShieldCheck aria-hidden="true" /> {text.secure}
          </p>
        </>
      )}
    </section>
  );
};
