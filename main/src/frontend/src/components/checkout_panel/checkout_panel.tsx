import { ArrowLeft, Check, CheckCircle2, CreditCard, FileText, FlaskConical, Infinity as InfinityIcon, Loader2, Lock, RefreshCw, ShieldCheck, Smartphone, Wallet, Zap } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Link, useHistory, useLocation } from "react-router-dom";

import { CheckoutTaxIdField } from "@/components/checkout_tax_id/checkout_tax_id_field";
import { CARD_LOGOS, MbWayLogo, PayPalLogo, StripeLogo } from "@/components/payment_logos/payment_logos";
import { StripeEmbeddedCheckout } from "@/components/stripe_embedded_checkout/stripe_embedded_checkout";
import { isValidOptionalTaxId } from "@/components/checkout_tax_id/tax_id";
import { useTestMode } from "@/components/test_mode/test_mode_context";
import { isValidTestEmail } from "@/components/test_mode/test_mode_email";
import { TestModeEmailField } from "@/components/test_mode/test_mode_email_field";
import { useProgramCheckout } from "@/hooks/use_program_checkout";
import { rememberAfterLogin } from "@/services/after_login";
import { receiptPath } from "@/services/invoices_api";
import { type Copy, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import type { DiscountPreview } from "@/services/discount_codes_api";

import { DiscountField } from "./discount_field";
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
  // The product picture for the order summary (optional).
  cover?: string;
  // Optional: run once when the purchase is confirmed (for an immediate
  // hand-off to the product's post-purchase experience).
  onSuccess?: () => void;
  // Only the pay button (and its states), for a page that already shows what
  // is bought and its price (v2.133.0); "ink" is the dark button for a light
  // panel.
  actionOnly?: boolean;
}

const COPY: Copy<{
  title: string;
  youGet: string;
  yourOrder: string;
  back: string;
  finish: string;
  secureVia: string;
  chooseMethod: string;
  stepDetails: string;
  stepBilling: string;
  trust: [string, string, string];
  acceptTerms: [string, string, string, string, string];
  waiveWithdrawal: string;
  consentRequired: string;
  order: string;
  total: string;
  oneTime: string;
  method: string;
  pay: (price: string) => string;
  payTest: string;
  testMode: string;
  methods: Record<string, { name: string; caption: string; detail: string }>;
  notConfigured: string;
  embeddedTitle: string;
  embeddedLoading: string;
  embeddedError: string;
  changeMethod: string;
  notConfiguredError: (name: string) => string;
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
    youGet: "What you get",
    yourOrder: "Your order",
    back: "Back",
    finish: "Checkout",
    secureVia: "Secure payment via",
    chooseMethod: "Choose how to pay",
    stepDetails: "Payment details",
    stepBilling: "Receipt",
    trust: ["Secure, encrypted payment", "Instant access", "One-time payment, no subscription"],
    acceptTerms: ["I have read and accept the ", "Terms and Conditions", " and the ", "Privacy Policy", "."],
    waiveWithdrawal: "I ask for immediate access to the digital content and acknowledge that, once access starts, I lose my 14-day right of withdrawal.",
    consentRequired: "Accept the terms and the immediate access to continue.",
    order: "Your order",
    total: "Total",
    oneTime: "One-time payment · no subscription",
    method: "Payment method",
    pay: (price) => `Pay ${price}`,
    payTest: "Complete Test Mode purchase",
    testMode: "Test Mode is active: the purchase completes without any payment provider.",
    methods: {
      card: { name: "Card", caption: "Visa, Mastercard, Amex", detail: "Enter the card number, expiry date and CVC on the next step, on the provider's secure page. The card type is recognised from the number." },
      mbway: { name: "MB WAY", caption: "Your phone", detail: "You get the request in the MB WAY app on your phone and confirm it there." },
      paypal: { name: "PayPal", caption: "Your PayPal account", detail: "You continue on PayPal and pay with your account or a card saved there." }
    },
    notConfigured: "Not set up",
    embeddedTitle: "Pay securely with Stripe",
    embeddedLoading: "Loading the secure payment form…",
    embeddedError: "The secure payment form could not be loaded. Check your connection and try again.",
    changeMethod: "Change payment method",
    notConfiguredError: (name) => `${name} is not set up in this environment yet, so this payment cannot be made. Nothing was charged.`,
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
    youGet: "O que recebes",
    yourOrder: "O teu pedido",
    back: "Voltar",
    finish: "Finalizar compra",
    secureVia: "Pagamento seguro via",
    chooseMethod: "Escolhe o método de pagamento",
    stepDetails: "Dados de pagamento",
    stepBilling: "Comprovativo",
    trust: ["Pagamento seguro e encriptado", "Acesso imediato", "Pagamento único, sem subscrição"],
    acceptTerms: ["Li e aceito os ", "Termos e Condições", " e a ", "Política de Privacidade", "."],
    waiveWithdrawal: "Peço o acesso imediato ao conteúdo digital e reconheço que, a partir desse momento, perco o direito de livre resolução de 14 dias.",
    consentRequired: "Aceita os termos e o acesso imediato para continuar.",
    order: "O teu pedido",
    total: "Total",
    oneTime: "Pagamento único · sem subscrição",
    method: "Método de pagamento",
    pay: (price) => `Pagar ${price}`,
    payTest: "Concluir compra em Modo de Teste",
    testMode: "O Modo de Teste está ativo: a compra conclui-se sem nenhum fornecedor de pagamento.",
    methods: {
      card: { name: "Cartão", caption: "Visa, Mastercard, Amex", detail: "Introduzes o número, a validade e o CVC no passo seguinte, na página segura do fornecedor. O tipo de cartão é reconhecido pelo número." },
      mbway: { name: "MB WAY", caption: "O teu telemóvel", detail: "Recebes o pedido na app MB WAY do teu telemóvel e confirmas lá." },
      paypal: { name: "PayPal", caption: "A tua conta PayPal", detail: "Continuas no PayPal e pagas com a tua conta ou com um cartão lá guardado." }
    },
    notConfigured: "Por configurar",
    embeddedTitle: "Pagamento seguro com Stripe",
    embeddedLoading: "A carregar o formulário de pagamento seguro…",
    embeddedError: "Não foi possível carregar o formulário de pagamento seguro. Verifica a ligação e tenta de novo.",
    changeMethod: "Trocar método de pagamento",
    notConfiguredError: (name) => `O pagamento por ${name} ainda não está configurado neste ambiente, por isso não é possível pagar. Nada foi cobrado.`,
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
      "You already have the other Premium level. Premium Level 1 and Level 2 cannot be active at the same time.": "Já tens o outro nível Premium. O Premium Nível 1 e o Nível 2 não podem estar ativos ao mesmo tempo.",
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

// Every payment method RYZE supports is shown (owner request, 2026-11-23: the
// whole screen must be visible even without providers). Which ones can be
// paid with still comes from the backend (/payments/methods); choosing one
// that is not set up explains it instead of starting a purchase.
const ALL_METHODS = ["card", "mbway", "paypal"] as const;
const CARD_BRANDS = ["VISA", "Mastercard", "AMEX", "Maestro"];

// CheckoutPanel is the payment step of every paid RYZE journey: the order
// summary with the fixed price, the payment methods the backend advertises
// (never hard-coded), the Test Mode shortcut and honest states when nothing
// can be charged. The purchase lifecycle is the shared useProgramCheckout
// hook; the backend stays authoritative for every outcome.
export const CheckoutPanel = ({ programId, itemTitle, itemCategory, priceMinor, includes, done, successNote, loginReturn, onSuccess, cover, actionOnly = false }: CheckoutPanelProps) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const { isActive: testModeActive } = useTestMode();
  const checkout = useProgramCheckout({ programId, free: false, testModeActive, enabled: Boolean(programId) });
  const money = new Intl.NumberFormat(locale === "pt" ? "pt-PT" : "en-IE", { style: "currency", currency: "EUR" });
  const formatPrice = (minor: number) => money.format(minor / 100);
  // A discount code checked on the pay screen lowers what is paid (v2.131.0).
  const [discount, setDiscount] = useState<DiscountPreview | null>(null);
  const listPrice = formatPrice(priceMinor);
  const price = discount ? formatPrice(discount.total_minor_units) : listPrice;
  const state = checkout.state;
  const busy = ["creating", "negotiating", "redirecting", "processing"].includes(state.status);
  const taxIdValid = isValidOptionalTaxId(checkout.taxId);
  const canPay = taxIdValid && (testModeActive ? isValidTestEmail(checkout.testEmail) : true);
  const available = new Set(checkout.paymentMethods.map((method) => method.method));
  const selected = checkout.selectedMethod || "card";
  const [methodError, setMethodError] = useState("");
  // EU consumer rules (Directive 2011/83/EU art. 16(m); DL 24/2014): the terms
  // are accepted and, for digital content supplied at once, the buyer asks for
  // immediate access and acknowledges losing the right of withdrawal.
  const [acceptedTerms, setAcceptedTerms] = useState(false);
  const [waived, setWaived] = useState(false);
  const consented = acceptedTerms && waived;
  useEffect(() => {
    checkout.setConsented(consented);
  }, [checkout, consented]);
  const pay = () => {
    if (!consented) {
      setMethodError(text.consentRequired);
      return;
    }
    if (!testModeActive && !available.has(selected)) {
      setMethodError(text.notConfiguredError(text.methods[selected]?.name ?? selected));
      return;
    }
    setMethodError("");
    void checkout.buy();
  };
  const translate = (message: string) => text.errors[message] ?? message;
  const successHandled = useRef(false);

  useEffect(() => {
    if (state.status === "success" && !successHandled.current) {
      successHandled.current = true;
      onSuccess?.();
    }
  }, [onSuccess, state.status]);

  // The payment happens on its own screen (owner request, 2026-11-23): the
  // page shows a simple summary and one Pay button; the screen ("?pagamento=1",
  // so the browser's back closes it) holds only the payment.
  const history = useHistory();
  const location = useLocation();
  const payOpen = new URLSearchParams(location.search).get("pagamento") === "1";
  const openPay = () => {
    const params = new URLSearchParams(location.search);
    params.set("pagamento", "1");
    history.push({ pathname: location.pathname, search: `?${params.toString()}` });
  };
  const closePay = () => {
    if (state.status === "embedded") checkout.backToMethods();
    const params = new URLSearchParams(location.search);
    params.delete("pagamento");
    const search = params.toString();
    history.replace({ pathname: location.pathname, search: search ? `?${search}` : "" });
  };

  useEffect(() => {
    if (!payOpen) return;
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previous;
    };
  }, [payOpen]);

  const doneView = (success: boolean) => (
    <div className={styles.doneView} aria-live="polite">
      <span className={styles.doneIcon} aria-hidden="true">
        <CheckCircle2 />
      </span>
      <h2 className={styles.doneTitle}>{success ? text.success : text.owned}</h2>
      <p className={styles.doneText}>{success ? text.successText : text.ownedText}</p>
      {successNote}
      <Link className={styles.primary} to={done.to}>
        {done.label}
      </Link>
      {success && state.status === "success" ? (
        <Link className={styles.secondary} to={receiptPath(state.purchase.id)}>
          <FileText aria-hidden="true" /> {text.receipt}
        </Link>
      ) : null}
    </div>
  );

  // The page block: what the member gets, the price and one button.
  const summary = (
    <section className={styles.summary} aria-labelledby="checkout-title">
      <h2 id="checkout-title" className={styles.summaryTitle}>
        {/* Without its own list (the page already shows what is included) the
            block is the order, not "what you get" again. */}
        {includes.length > 0 ? text.youGet : text.yourOrder}
      </h2>
      <p className={styles.summaryItem}>
        <strong>{itemTitle}</strong>
        <span>{itemCategory}</span>
      </p>
      {includes.length > 0 ? (
        <ul className={styles.summaryList}>
          {includes.map((line) => (
            <li key={line}>
              <CheckCircle2 aria-hidden="true" />
              {line}
            </li>
          ))}
        </ul>
      ) : null}
      <div className={styles.summaryPrice}>
        <strong>{price}</strong>
        <span>{text.oneTime}</span>
      </div>
      {state.status === "checking" ? (
        <p className={styles.status}>
          <Loader2 className={styles.spin} aria-hidden="true" /> {text.checking}
        </p>
      ) : state.status === "guest" ? (
        <>
          <p className={styles.summaryNote}>{text.guestText}</p>
          <Link className={styles.payButton} to="/login" onClick={() => rememberAfterLogin(loginReturn)}>
            {text.login}
          </Link>
        </>
      ) : (
        <button type="button" className={styles.payButton} onClick={openPay}>
          <Lock aria-hidden="true" /> {text.pay(price)}
        </button>
      )}
    </section>
  );

  if (state.status === "owned" || (state.status === "success" && !payOpen)) {
    return <section className={styles.summary}>{doneView(state.status === "success")}</section>;
  }

  const methodMark = (method: string) =>
    method === "card" ? (
      <span className={styles.markRow}>
        {CARD_LOGOS.slice(0, 3).map((Logo, index) => (
          <span key={index} className={styles.mark}>
            <Logo />
          </span>
        ))}
      </span>
    ) : method === "mbway" ? (
      <span className={styles.markWide}>
        <MbWayLogo />
      </span>
    ) : (
      <span className={styles.markWide}>
        <PayPalLogo />
      </span>
    );

  const consents = (
    <div className={styles.consents}>
      <label className={styles.consent}>
        <input type="checkbox" checked={acceptedTerms} onChange={(event) => setAcceptedTerms(event.target.checked)} />
        <span>
          {text.acceptTerms[0]}
          <Link to="/legal/termos" target="_blank" rel="noopener">
            {text.acceptTerms[1]}
          </Link>
          {text.acceptTerms[2]}
          <Link to="/legal/privacidade" target="_blank" rel="noopener">
            {text.acceptTerms[3]}
          </Link>
          {text.acceptTerms[4]}
        </span>
      </label>
      <label className={styles.consent}>
        <input type="checkbox" checked={waived} onChange={(event) => setWaived(event.target.checked)} />
        <span>{text.waiveWithdrawal}</span>
      </label>
    </div>
  );

  const payButton = (
    <button type="button" className={styles.payButton} disabled={!canPay || busy} onClick={pay}>
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
  );

  const payment =
    state.status === "cancelled" ? (
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
        {testModeActive ? <TestModeEmailField value={checkout.testEmail} onChange={checkout.setTestEmail} /> : null}
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
          <div className={styles.step}>
            <p className={styles.testMode}>
              <FlaskConical aria-hidden="true" /> {text.testMode}
            </p>
            <TestModeEmailField value={checkout.testEmail} onChange={checkout.setTestEmail} />
          </div>
        ) : (
          <fieldset className={styles.step} disabled={state.status === "embedded"}>
            <legend className={styles.stepTitle}>
              <span>1</span> {text.chooseMethod}
            </legend>
            <div className={styles.methodCards}>
              {ALL_METHODS.map((method) => {
                const Icon = METHOD_ICON[method] ?? CreditCard;
                const copy = text.methods[method];
                const active = selected === method;
                return (
                  <label key={method} className={joinClassNames(styles.methodCard, active && styles.methodCardActive)}>
                    <input
                      type="radio"
                      name="payment-method"
                      value={method}
                      checked={active}
                      onChange={() => {
                        checkout.selectMethod(method);
                        setMethodError("");
                      }}
                    />
                    <span className={styles.methodCardTop}>
                      <span className={styles.tileRadio} aria-hidden="true">
                        {active ? <Check /> : null}
                      </span>
                      <span className={styles.methodCardName}>
                        <Icon aria-hidden="true" /> {copy.name}
                      </span>
                    </span>
                    {methodMark(method)}
                    <span className={styles.methodCardCaption}>{copy.caption}</span>
                    {available.has(method) ? null : <span className={styles.tileBadge}>{text.notConfigured}</span>}
                  </label>
                );
              })}
            </div>
          </fieldset>
        )}

        {state.status === "embedded" ? (
          <div className={styles.step}>
            <p className={styles.stepTitle}>
              <span>2</span> {text.stepDetails}
            </p>
            <StripeEmbeddedCheckout
              publishableKey={state.publishableKey}
              clientSecret={state.clientSecret}
              loadingLabel={text.embeddedLoading}
              errorLabel={text.embeddedError}
            />
            <button type="button" className={styles.secondary} onClick={checkout.backToMethods}>
              {text.changeMethod}
            </button>
          </div>
        ) : (
          <>
            {testModeActive ? null : (
              <div className={styles.step}>
                <p className={styles.stepTitle}>
                  <span>2</span> {text.stepDetails}
                </p>
                <div className={styles.detailCard}>
                  {selected === "card" ? (
                    <div className={styles.cardArt} aria-hidden="true">
                      <span className={styles.cardChip} />
                      <span className={styles.cardNumber}>•••• •••• •••• ••••</span>
                      <span className={styles.cardBrands}>
                        {CARD_LOGOS.map((Logo, index) => (
                          <span key={index} className={styles.mark}>
                            <Logo />
                          </span>
                        ))}
                      </span>
                    </div>
                  ) : (
                    <div className={styles.detailMark} aria-hidden="true">
                      {selected === "mbway" ? <MbWayLogo /> : <PayPalLogo />}
                    </div>
                  )}
                  <p>{text.methods[selected]?.detail}</p>
                </div>
              </div>
            )}
            <div className={styles.step}>
              <p className={styles.stepTitle}>
                <span>{testModeActive ? 1 : 3}</span> {text.stepBilling}
              </p>
              <CheckoutTaxIdField value={checkout.taxId} onChange={checkout.setTaxId} disabled={busy} />
              {testModeActive ? null : (
                <DiscountField
                  programId={programId}
                  disabled={busy}
                  formatPrice={formatPrice}
                  onChange={(preview) => {
                    setDiscount(preview);
                    checkout.setDiscountCode(preview?.code ?? "");
                  }}
                />
              )}
            </div>
            {consents}
            {methodError ? (
              <p className={joinClassNames(styles.notice, styles.noticeError, styles.methodError)} role="alert">
                {methodError}
              </p>
            ) : null}
            {payButton}
          </>
        )}
        <p className={styles.secure}>
          <ShieldCheck aria-hidden="true" /> {text.secure}
        </p>
      </>
    );

  const action = (
    <div className={styles.actionOnly}>
      {state.status === "checking" ? (
        <p className={styles.status}>
          <Loader2 className={styles.spin} aria-hidden="true" /> {text.checking}
        </p>
      ) : state.status === "guest" ? (
        <>
          <Link className={joinClassNames(styles.payButton, styles.payButtonInk)} to="/login" onClick={() => rememberAfterLogin(loginReturn)}>
            {text.login}
          </Link>
          <p className={styles.actionNote}>{text.guestText}</p>
        </>
      ) : (
        <button type="button" className={joinClassNames(styles.payButton, styles.payButtonInk)} onClick={openPay}>
          <Lock aria-hidden="true" /> {text.pay(price)}
        </button>
      )}
    </div>
  );

  return (
    <>
      {actionOnly ? action : summary}
      {payOpen
        ? createPortal(
            <div className={styles.screen} role="dialog" aria-modal="true" aria-labelledby="checkout-finish">
              <header className={styles.screenBar}>
                <button type="button" className={styles.backButton} onClick={closePay}>
                  <ArrowLeft aria-hidden="true" /> {text.back}
                </button>
                <span className={styles.screenBrand}>RYZE</span>
                <span className={styles.screenLock}>
                  <Lock aria-hidden="true" />
                </span>
              </header>
              <div className={styles.screenBody}>
                <aside className={styles.orderCard} aria-label={text.order}>
                  <div className={styles.orderHero}>
                    {cover ? <img src={cover} alt="" /> : <span className={styles.orderHeroFallback} />}
                    <span className={styles.orderHeroText}>
                      <em>{itemCategory}</em>
                      <strong>{itemTitle}</strong>
                    </span>
                  </div>
                  <div className={styles.orderBody}>
                    <h2 className={styles.orderHeading}>{text.order}</h2>
                    {includes.length > 0 ? (
                      <ul className={styles.summaryList}>
                        {includes.map((line) => (
                          <li key={line}>
                            <CheckCircle2 aria-hidden="true" />
                            {line}
                          </li>
                        ))}
                      </ul>
                    ) : null}
                    <div className={styles.orderRows}>
                      <span>{itemTitle}</span>
                      <span>{price}</span>
                    </div>
                    <div className={styles.orderTotal}>
                      <span>{text.total}</span>
                      <strong>
                        {discount ? <s className={styles.listPrice}>{listPrice}</s> : null} {price}
                      </strong>
                    </div>
                    <ul className={styles.trust}>
                      <li>
                        <ShieldCheck aria-hidden="true" /> {text.trust[0]}
                      </li>
                      <li>
                        <Zap aria-hidden="true" /> {text.trust[1]}
                      </li>
                      <li>
                        <InfinityIcon aria-hidden="true" /> {text.trust[2]}
                      </li>
                    </ul>
                  </div>
                </aside>
                <section className={styles.payCard}>
                  {state.status === "success" ? (
                    doneView(true)
                  ) : (
                    <>
                      <header className={styles.payHead}>
                        <h1 id="checkout-finish" className={styles.finishTitle}>
                          {text.finish}
                        </h1>
                        <p className={styles.secureVia}>
                          <Lock aria-hidden="true" /> {text.secureVia}
                          <span className={styles.stripeMark}>
                            <StripeLogo />
                          </span>
                        </p>
                      </header>
                      {payment}
                    </>
                  )}
                </section>
              </div>
            </div>,
            document.body
          )
        : null}
    </>
  );
};
