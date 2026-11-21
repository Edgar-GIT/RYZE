import { CheckCircle2, Clock3, Inbox, XCircle } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useHistory } from "react-router-dom";

import { useI18n } from "@/i18n/i18n";
import {
  acceptPremiumLevel2Offer,
  declinePremiumLevel2Offer,
  fetchTrainerPremiumLevel2Offers,
  setPremiumLevel2Availability,
  type PremiumLevel2Offers
} from "@/services/premium_level_2_api";
import { ApiError } from "@utils/http_client";

import styles from "./premium_level2_offers_panel.module.css";

const COPY = {
  en: {
    title: "Requests waiting for your answer",
    lead: "RYZE offers each new Premium Level 2 request to one available trainer at a time. You have 24 hours to accept or decline; then it goes to the next trainer. Before accepting you only see the service and the dates; the intake opens once you accept.",
    toggle: "I take Premium Level 2 requests",
    toggleHint: "Turn it off when you cannot take new clients; requests you already accepted are not affected.",
    none: "No request is waiting for you.",
    requested: (date: string) => `Requested on ${date}`,
    deadline: (left: string) => `Answer ${left}`,
    accept: "Accept",
    decline: "Decline",
    expired: "The 24 hours ended: the request went to another trainer.",
    gone: "This request is no longer available.",
    error: "Something went wrong. Please try again.",
    loadError: "We could not load your offers."
  },
  pt: {
    title: "Pedidos à espera da tua resposta",
    lead: "A RYZE oferece cada novo pedido Premium Nível 2 a um treinador disponível de cada vez. Tens 24 horas para aceitar ou recusar; depois passa ao treinador seguinte. Antes de aceitares só vês o serviço e as datas; o questionário abre quando aceitas.",
    toggle: "Aceito pedidos Premium Nível 2",
    toggleHint: "Desliga quando não puderes receber novos clientes; os pedidos que já aceitaste não mudam.",
    none: "Não há nenhum pedido à tua espera.",
    requested: (date: string) => `Pedido a ${date}`,
    deadline: (left: string) => `Responde ${left}`,
    accept: "Aceitar",
    decline: "Recusar",
    expired: "As 24 horas terminaram: o pedido passou a outro treinador.",
    gone: "Este pedido já não está disponível.",
    error: "Algo correu mal. Tenta de novo.",
    loadError: "Não foi possível carregar os teus pedidos."
  }
} as const;

// PremiumLevel2OffersPanel is the trainer's side of allocation by offer
// (owner decision, 2026-11-21): the availability switch and the requests
// offered to them, to accept or decline.
export const PremiumLevel2OffersPanel = () => {
  const { locale, intlLocale, formatDate } = useI18n();
  const text = COPY[locale];
  const history = useHistory();
  const [data, setData] = useState<PremiumLevel2Offers | null>(null);
  const [busy, setBusy] = useState("");
  const [message, setMessage] = useState("");

  const load = useCallback(async () => {
    try {
      setData(await fetchTrainerPremiumLevel2Offers());
    } catch {
      setMessage(text.loadError);
    }
  }, [text.loadError]);

  useEffect(() => {
    void load();
  }, [load]);

  const errorText = (error: unknown) => {
    const code = error instanceof ApiError ? error.code : "";
    return code === "OFFER_EXPIRED" ? text.expired : code === "OFFER_NOT_FOUND" || code === "REQUEST_NOT_AVAILABLE" ? text.gone : text.error;
  };

  const toggle = async (accepts: boolean) => {
    setBusy("toggle");
    setMessage("");
    try {
      await setPremiumLevel2Availability(accepts);
      await load();
    } catch (error) {
      setMessage(errorText(error));
    } finally {
      setBusy("");
    }
  };

  const answer = async (offerId: string, accept: boolean) => {
    setBusy(offerId);
    setMessage("");
    try {
      if (accept) {
        const result = await acceptPremiumLevel2Offer(offerId);
        history.push(`/trainer/premium-level-2/${encodeURIComponent(result.request_id)}`);
        return;
      }
      await declinePremiumLevel2Offer(offerId);
      await load();
    } catch (error) {
      setMessage(errorText(error));
      await load();
    } finally {
      setBusy("");
    }
  };

  const left = (iso: string) => {
    const hours = Math.max(0, (new Date(iso).getTime() - Date.now()) / 3_600_000);
    const format = new Intl.RelativeTimeFormat(intlLocale, { numeric: "auto" });
    return hours < 1 ? format.format(Math.max(1, Math.round(hours * 60)), "minute") : format.format(Math.round(hours), "hour");
  };

  return (
    <section className={styles.panel} aria-labelledby="pl2-offers-title">
      <div className={styles.head}>
        <h2 id="pl2-offers-title" className={styles.title}>
          <Inbox aria-hidden="true" /> {text.title}
        </h2>
        <p className={styles.lead}>{text.lead}</p>
      </div>

      <label className={styles.toggle}>
        <input
          type="checkbox"
          role="switch"
          checked={Boolean(data?.accepts_requests)}
          disabled={!data || busy === "toggle"}
          onChange={(event) => void toggle(event.target.checked)}
        />
        <span>
          <strong>{text.toggle}</strong>
          <small>{text.toggleHint}</small>
        </span>
      </label>

      {message ? (
        <p className={styles.message} role="alert">
          {message}
        </p>
      ) : null}

      {data && data.offers.length === 0 ? <p className={styles.muted}>{text.none}</p> : null}
      {data && data.offers.length > 0 ? (
        <ul className={styles.offers}>
          {data.offers.map((offer) => (
            <li key={offer.id} className={styles.offer}>
              <div className={styles.offerText}>
                <strong>{offer.offer_name || "Premium Level 2"}</strong>
                <span>{text.requested(formatDate(offer.request_created_at, { day: "numeric", month: "short" }))}</span>
                <span className={styles.deadline}>
                  <Clock3 aria-hidden="true" /> {text.deadline(left(offer.expires_at))}
                </span>
              </div>
              <div className={styles.buttons}>
                <button type="button" className={styles.accept} disabled={Boolean(busy)} onClick={() => void answer(offer.id, true)}>
                  <CheckCircle2 aria-hidden="true" /> {text.accept}
                </button>
                <button type="button" className={styles.decline} disabled={Boolean(busy)} onClick={() => void answer(offer.id, false)}>
                  <XCircle aria-hidden="true" /> {text.decline}
                </button>
              </div>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
};
