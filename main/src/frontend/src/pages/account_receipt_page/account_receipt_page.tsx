import { ArrowLeft, FlaskConical, Printer, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useHistory, useParams } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import { Button } from "@/components/button/button";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useI18n } from "@/i18n/i18n";
import { fetchPurchaseInvoice, type PurchaseInvoice } from "@/services/invoices_api";
import { ApiError } from "@utils/http_client";

import styles from "./account_receipt_page.module.css";

type State = { status: "loading" } | { status: "error" } | { status: "missing" } | { status: "ready"; invoice: PurchaseInvoice };

const COPY = {
  en: {
    back: "Purchases",
    print: "Print or save as PDF",
    title: "Purchase receipt",
    number: "Number",
    issued: "Issue date",
    purchased: "Purchase date",
    seller: "Seller",
    customer: "Customer",
    taxId: "Taxpayer no.",
    consumer: "Final consumer",
    item: "Item",
    qty: "Qty",
    unit: "Unit price",
    vat: "VAT",
    total: "Total",
    method: "Payment method",
    methods: { card: "Card", mbway: "MB WAY", paypal: "PayPal" } as Record<string, string>,
    exemption: { M10: "VAT exempt under article 53 of the Portuguese VAT Code (small-business scheme)." } as Record<string, string>,
    test: "Test Mode purchase: no payment was taken.",
    noTax: "This document is a purchase receipt and is not a tax invoice.",
    missing: "We could not find a receipt for this purchase.",
    missingText: "Receipts are issued for completed purchases. Check your purchases list.",
    loadError: "We could not load this receipt.",
    retry: "Try again"
  },
  pt: {
    back: "Compras",
    print: "Imprimir ou guardar em PDF",
    title: "Comprovativo de compra",
    number: "Número",
    issued: "Data de emissão",
    purchased: "Data da compra",
    seller: "Vendedor",
    customer: "Cliente",
    taxId: "NIF",
    consumer: "Consumidor final",
    item: "Artigo",
    qty: "Qtd.",
    unit: "Preço unitário",
    vat: "IVA",
    total: "Total",
    method: "Método de pagamento",
    methods: { card: "Cartão", mbway: "MB WAY", paypal: "PayPal" } as Record<string, string>,
    exemption: { M10: "IVA - regime de isenção (artigo 53.º do CIVA)." } as Record<string, string>,
    test: "Compra do Modo de Teste: não foi cobrado nenhum pagamento.",
    noTax: "Este documento é um comprovativo de compra e não tem valor fiscal.",
    missing: "Não encontrámos o comprovativo desta compra.",
    missingText: "Os comprovativos são emitidos para compras concluídas. Confirma a tua lista de compras.",
    loadError: "Não foi possível carregar este comprovativo.",
    retry: "Tentar de novo"
  }
} as const;

// AccountReceiptPage shows the purchase document of one of the member's
// purchases (the backend serves only the owner's), with a print layout so
// the browser's "Save as PDF" produces the file. It is a purchase receipt with
// no tax value until certified invoicing software is connected.
export const AccountReceiptPage = () => {
  const { purchaseId } = useParams<{ purchaseId: string }>();
  const { locale, formatDate, formatMoney } = useI18n();
  const text = COPY[locale];
  const history = useHistory();
  const [state, setState] = useState<State>({ status: "loading" });

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      setState({ status: "ready", invoice: await fetchPurchaseInvoice(purchaseId) });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      setState(error instanceof ApiError && error.status === 404 ? { status: "missing" } : { status: "error" });
    }
  }, [history, purchaseId]);

  useEffect(() => {
    void load();
  }, [load]);

  // Printing shows only the document (see the module's print rules).
  useEffect(() => {
    document.body.classList.add("ryze-receipt-print");
    return () => document.body.classList.remove("ryze-receipt-print");
  }, []);

  const date = (value: string) => formatDate(value, { year: "numeric", month: "long", day: "numeric" });

  return (
    <PageWrapper className={styles.page}>
      <div className={styles.toolbar}>
        <Button to="/account/purchases" variant="ghost" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
          {text.back}
        </Button>
        {state.status === "ready" ? (
          <Button variant="secondary" size="small" onClick={() => window.print()} icon={<Printer size={15} />} iconPosition="left">
            {text.print}
          </Button>
        ) : null}
      </div>

      {state.status === "loading" ? <AppLoading /> : null}

      {state.status === "missing" ? (
        <section className={styles.stateCard}>
          <p className={styles.stateTitle}>{text.missing}</p>
          <p>{text.missingText}</p>
        </section>
      ) : null}

      {state.status === "error" ? (
        <section className={styles.stateCard} role="alert">
          <p>{text.loadError}</p>
          <Button variant="secondary" size="small" onClick={() => void load()} icon={<RefreshCw size={15} />}>
            {text.retry}
          </Button>
        </section>
      ) : null}

      {state.status === "ready" ? <Receipt invoice={state.invoice} text={text} date={date} money={formatMoney} /> : null}
    </PageWrapper>
  );
};

interface ReceiptProps {
  invoice: PurchaseInvoice;
  text: (typeof COPY)[keyof typeof COPY];
  date: (value: string) => string;
  money: (minorUnits: number, currency: string) => string;
}

const Receipt = ({ invoice, text, date, money }: ReceiptProps) => (
  <article className={styles.document} aria-labelledby="receipt-title">
    <header className={styles.documentHead}>
      <div>
        <p className={styles.brand}>RYZE</p>
        <h1 id="receipt-title" className={styles.documentTitle}>
          {text.title}
        </h1>
      </div>
      <dl className={styles.facts}>
        <div>
          <dt>{text.number}</dt>
          <dd className={styles.number}>{invoice.number}</dd>
        </div>
        <div>
          <dt>{text.issued}</dt>
          <dd>{date(invoice.issued_at)}</dd>
        </div>
        <div>
          <dt>{text.purchased}</dt>
          <dd>{date(invoice.purchased_at)}</dd>
        </div>
      </dl>
    </header>

    {invoice.test ? (
      <p className={styles.testNote}>
        <FlaskConical size={14} aria-hidden="true" /> {text.test}
      </p>
    ) : null}

    <div className={styles.parties}>
      <section>
        <h2>{text.seller}</h2>
        <p>{invoice.seller_name}</p>
        {invoice.seller_tax_id ? <p>{`${text.taxId} ${invoice.seller_tax_id}`}</p> : null}
        {invoice.seller_address ? <p>{invoice.seller_address}</p> : null}
      </section>
      <section>
        <h2>{text.customer}</h2>
        <p>{invoice.buyer_name}</p>
        <p>{`${text.taxId} ${invoice.buyer_tax_id ?? text.consumer}`}</p>
      </section>
    </div>

    <table className={styles.lines}>
      <thead>
        <tr>
          <th scope="col">{text.item}</th>
          <th scope="col">{text.qty}</th>
          <th scope="col">{text.unit}</th>
          <th scope="col">{text.vat}</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>{invoice.description}</td>
          <td>{invoice.quantity}</td>
          <td>{money(invoice.unit_price_minor_units, invoice.currency)}</td>
          <td>{money(invoice.tax_minor_units, invoice.currency)}</td>
        </tr>
      </tbody>
    </table>

    <dl className={styles.totals}>
      <div>
        <dt>{text.vat}</dt>
        <dd>{money(invoice.tax_minor_units, invoice.currency)}</dd>
      </div>
      <div className={styles.grandTotal}>
        <dt>{text.total}</dt>
        <dd>{money(invoice.total_minor_units, invoice.currency)}</dd>
      </div>
      {invoice.payment_method ? (
        <div>
          <dt>{text.method}</dt>
          <dd>{text.methods[invoice.payment_method] ?? invoice.payment_method}</dd>
        </div>
      ) : null}
    </dl>

    <footer className={styles.notes}>
      {invoice.tax_exemption_code && text.exemption[invoice.tax_exemption_code] ? <p>{text.exemption[invoice.tax_exemption_code]}</p> : null}
      <p>{text.noTax}</p>
    </footer>
  </article>
);
