import { ArrowLeft, CheckCircle2, Sparkles } from "lucide-react";
import { useEffect, useState } from "react";
import { useHistory, useLocation } from "react-router-dom";

import { CheckoutPanel } from "@/components/checkout_panel/checkout_panel";
import j from "@/components/journey/journey.module.css";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import ui from "@/components/public_ui/public_ui.module.css";

import { type Copy, useI18n } from "@/i18n/i18n";
import { formatMarketplacePrice, type MarketplaceProgram } from "@/services/marketplace_api";
import { fetchPremiumLevel2Programs } from "@/services/premium_level_2_api";

const COPY: Copy<{
  backServices: string;
  steps: string[];
  kicker: string;
  title: string;
  titleAccent: string;
  lead: string;
  panelTitle: string;
  priceNote: string;
  includes: string[];
  how: { title: string; text: string }[];
  unavailable: string;
  unavailableText: string;
  category: string;
  done: string;
  successNote: string;
}> = {
  en: {
    backServices: "Services",
    steps: ["Service", "Payment"],
    kicker: "RYZE Personalised",
    title: "Training and nutrition ",
    titleAccent: "made only for you.",
    lead: "The most complete RYZE service. Buy once and your detailed questionnaire opens immediately. We analyse your answers and a RYZE trainer prepares your training and nutrition plans, fully adapted to you.",
    panelTitle: "What you get",
    priceNote: "one-time",
    includes: [
      "A full questionnaire about you",
      "Training plan 100% personalised",
      "Nutrition plan adapted to your routine",
      "Prepared and reviewed by a RYZE trainer"
    ],
    how: [
      { title: "Buy the service", text: "One payment. No subscription." },
      { title: "Complete your questionnaire", text: "It opens right after payment, in your language." },
      { title: "We prepare your plan", text: "A RYZE trainer builds your training and nutrition." },
      { title: "Get notified", text: "You are alerted in the app and by email when it is ready." }
    ],
    unavailable: "Personalised is not open yet",
    unavailableText: "No Personalised offer is available right now. Please come back soon.",
    category: "Personalised · training + nutrition",
    done: "Start my experience",
    successNote: "Payment confirmed. Your questionnaire is ready."
  },
  pt: {
    backServices: "Serviços",
    steps: ["Serviço", "Pagamento"],
    kicker: "RYZE Personalizado",
    title: "Treino e nutrição ",
    titleAccent: "feitos só para ti.",
    lead: "O serviço mais completo da RYZE. Pagas uma única vez e o teu questionário detalhado abre logo de seguida. Analisamos as respostas e um treinador RYZE prepara os teus planos de treino e nutrição, totalmente adaptados a ti.",
    panelTitle: "O que recebes",
    priceNote: "pagamento único",
    includes: [
      "Questionário completo sobre ti",
      "Plano de treino 100% personalizado",
      "Plano de nutrição adaptado à tua rotina",
      "Preparado e revisto por um treinador RYZE"
    ],
    how: [
      { title: "Compra o serviço", text: "Um pagamento. Sem subscrição." },
      { title: "Preenche o questionário", text: "Abre logo após o pagamento, no teu idioma." },
      { title: "Preparamos o teu plano", text: "Um treinador RYZE cria o teu treino e a tua nutrição." },
      { title: "Recebes a notificação", text: "Avisamos-te na app e por email quando estiver pronto." }
    ],
    unavailable: "O Personalizado ainda não está disponível",
    unavailableText: "De momento não há nenhuma oferta Personalizado disponível. Volta em breve.",
    category: "Personalizado · treino + nutrição",
    done: "Começar a minha experiência",
    successNote: "Pagamento confirmado. O teu questionário está pronto."
  }
};

type Load = { status: "loading" } | { status: "ready" } | { status: "unavailable" };

// PremiumLevel2JourneyPage sells the trainer-delivered service. The long
// questionnaire is no longer a checkout prerequisite (owner request,
// 2026-11-15): payment comes first and hands the buyer to the post-purchase
// experience, where the questionnaire opens immediately.
export const PremiumLevel2JourneyPage = () => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const history = useHistory();
  const location = useLocation();
  const offerParam = new URLSearchParams(location.search).get("offer");
  const [offer, setOffer] = useState<MarketplaceProgram | null>(null);
  const [load, setLoad] = useState<Load>({ status: "loading" });

  useEffect(() => {
    let cancelled = false;
    fetchPremiumLevel2Programs()
      .then((programs) => {
        if (cancelled) return;
        setOffer(programs.find((program) => program.id === offerParam) ?? programs[0] ?? null);
        setLoad({ status: programs.length > 0 ? "ready" : "unavailable" });
      })
      .catch(() => {
        if (!cancelled) setLoad({ status: "unavailable" });
      });
    return () => {
      cancelled = true;
    };
  }, [offerParam]);

  useEffect(() => {
    window.scrollTo({ top: 0, behavior: "smooth" });
  }, []);

  return (
    <PageWrapper className={ui.page}>
      <div className={j.page}>
        <div className={j.top}>
          <button type="button" className={j.back} onClick={() => history.push("/services")}>
            <ArrowLeft aria-hidden="true" /> {text.backServices}
          </button>
          <ol className={j.progress} aria-label={text.steps[1]}>
            {text.steps.map((label, index) => (
              <li key={label} data-state={index === 0 ? "done" : "current"}>
                <span className={j.progressLabel}>{label}</span>
              </li>
            ))}
          </ol>
        </div>

        <div className={j.hero}>
          <div>
            <p className={j.kicker}>
              <Sparkles aria-hidden="true" /> {text.kicker}
            </p>
            <h1 className={j.title}>
              {text.title}
              <span>{text.titleAccent}</span>
            </h1>
            <p className={j.lead}>{text.lead}</p>
            {load.status === "unavailable" || !offer ? (
              <div className={j.notice}>
                <strong>{text.unavailable}</strong>
                <p>{text.unavailableText}</p>
              </div>
            ) : null}
          </div>
          <aside className={j.heroPanel} aria-label={text.panelTitle}>
            <h2>{text.panelTitle}</h2>
            <p className={j.heroPrice}>
              {offer ? formatMarketplacePrice(offer.price_minor_units, offer.currency, offer.type, locale) : "—"}
              <span>{text.priceNote}</span>
            </p>
            <ul className={j.heroList}>
              {text.includes.map((line) => (
                <li key={line}>
                  <CheckCircle2 aria-hidden="true" />
                  {line}
                </li>
              ))}
            </ul>
            {/* The pay button lives in this block (owner request, 2026-11-28):
                it already says what is bought and the price. */}
            {offer && load.status === "ready" ? (
              <CheckoutPanel
                actionOnly
                programId={offer.id}
                itemTitle={offer.name}
                itemCategory={text.category}
                priceMinor={offer.price_minor_units}
                includes={[]}
                done={{ to: "/services/premium-level-2/experience", label: text.done }}
                successNote={<p>{text.successNote}</p>}
                loginReturn={`/services/premium-level-2?offer=${encodeURIComponent(offer.id)}`}
                onSuccess={() => history.replace("/services/premium-level-2/experience?preparing=1")}
              />
            ) : null}
          </aside>
        </div>

        <ol className={j.steps}>
          {text.how.map((item) => (
            <li key={item.title}>
              <div>
                <strong>{item.title}</strong>
                <span>{item.text}</span>
              </div>
            </li>
          ))}
        </ol>

      </div>
    </PageWrapper>
  );
};

export default PremiumLevel2JourneyPage;
