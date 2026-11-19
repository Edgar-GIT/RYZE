import {
  ArrowLeft,
  ArrowRight,
  Bell,
  BookOpen,
  Check,
  CheckCircle2,
  ClipboardList,
  Clock3,
  Download,
  Dumbbell,
  FileText,
  Loader2,
  Mail,
  RefreshCw,
  ShieldCheck,
  Sparkles
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useHistory, useLocation, useParams } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { QuestionField } from "@/components/premium_level2_intake/question_field";
import ui from "@/components/public_ui/public_ui.module.css";
import { COMMERCE_PT } from "@/i18n/catalogs/commerce.pt";
import { PREMIUM_LEVEL2_PT } from "@/i18n/catalogs/premium_level2.pt";
import { QUESTIONNAIRE_PT } from "@/i18n/catalogs/questionnaire.pt";
import { type Copy, useGettext, useI18n } from "@/i18n/i18n";
import {
  fetchMyPremiumLevel2Request,
  fetchMyPremiumLevel2Requests,
  fetchPremiumLevel2Catalog,
  submitPremiumLevel2Intake,
  type PremiumLevel2Catalog,
  type PremiumLevel2ClientDetail,
  type PremiumLevel2Question
} from "@/services/premium_level_2_api";
import {
  isEmptyPremiumLevel2Answer,
  type PremiumLevel2AnswerValue
} from "@/services/premium_level_2_format";
import { fetchMyPurchases, type PurchaseHistoryEntry } from "@/services/purchases_api";
import { ApiError } from "@utils/http_client";

import styles from "./premium_level2_experience_page.module.css";

type AnswerValue = PremiumLevel2AnswerValue;
type Answers = Record<string, AnswerValue>;
type View = "preparing" | "questionnaire" | "soon" | "summary";

const EDITABLE_STATUSES = new Set(["awaiting_intake", "awaiting_trainer_assignment"]);

const COPY: Copy<{
  loading: string;
  errorTitle: string;
  errorText: string;
  retry: string;
  back: string;
  preparingKicker: string;
  preparingTitle: string;
  preparingLead: string;
  preparingSteps: string[];
  preparingDone: string;
  preparingButton: string;
  questionnaireKicker: string;
  questionnaireTitle: string;
  questionnaireLead: string;
  step: (current: number, total: number) => string;
  previous: string;
  next: string;
  submit: string;
  submitting: string;
  sectionProgress: (done: number, total: number) => string;
  required: string;
  fix: string;
  notice: string;
  soonKicker: string;
  soonTitle: string;
  soonLead: string;
  soonSubmitted: string;
  soonTrainer: string;
  soonBuilding: string;
  soonNotification: string;
  soonTrainerPending: string;
  soonTrainerNamed: (name: string) => string;
  soonBuildingText: string;
  soonNotificationText: string;
  viewSummary: string;
  editAnswers: string;
  viewService: string;
  deliveredTitle: string;
  deliveredText: string;
  openTraining: string;
  openAcademy: string;
  summaryKicker: string;
  summaryTitle: string;
  summaryLead: string;
  analysisTitle: string;
  analysisNote: string;
  analysisGoal: string;
  analysisTraining: string;
  analysisNutrition: string;
  analysisRecovery: string;
  analysisHealth: string;
  analysisNext: string;
  flagMedical: string;
  flagInjury: string;
  flagMedication: string;
  flagMovement: string;
  flagClearance: string;
  flagSmoking: string;
  noHealthFlags: string;
  answersTitle: string;
  answersLead: string;
  showAnswers: string;
  receiptTitle: string;
  receiptReference: string;
  receiptDate: string;
  receiptAmount: string;
  receiptMethod: string;
  receiptStatus: string;
  receiptTest: string;
  receiptTestText: string;
  receiptPending: string;
  receiptStatusCompleted: string;
  download: string;
  downloadNote: string;
  printTitle: string;
  printAnswers: string;
  printReceipt: string;
  printFooter: string;
  notAnswered: string;
  yes: string;
  no: string;
  status: string;
  planPending: string;
  planPendingText: string;
  serviceLink: string;
  purchaseFallback: string;
  emailNote: string;
  sessions: (count: number) => string;
  minutes: (count: number) => string;
  meals: (count: number) => string;
  sleep: (hours: string) => string;
}> = {
  en: {
    loading: "Loading your Premium Level 2 experience…",
    errorTitle: "We could not open your experience",
    errorText: "Please try again. Your purchase and answers are safe.",
    retry: "Try again",
    back: "Premium Level 2",
    preparingKicker: "Purchase confirmed",
    preparingTitle: "We are preparing your complete experience",
    preparingLead: "Your payment is confirmed. Next comes the questionnaire your RYZE trainer will use to build your individual plan.",
    preparingSteps: ["Payment confirmed", "Access linked to your account", "Questionnaire ready"],
    preparingDone: "Everything is ready",
    preparingButton: "Start my questionnaire",
    questionnaireKicker: "Your Premium Level 2",
    questionnaireTitle: "Tell us about you",
    questionnaireLead: "Answer honestly and completely. Your trainer uses every detail to build your training and nutrition around your life.",
    step: (current, total) => `Section ${current} of ${total}`,
    previous: "Previous",
    next: "Next",
    submit: "Send my answers",
    submitting: "Sending…",
    sectionProgress: (done, total) => `${done} of ${total} sections reviewed`,
    required: "This answer is required.",
    fix: "Complete the highlighted answers to continue.",
    notice: "Your answers stay in this tab until you send them. Do not close the page before finishing.",
    soonKicker: "Questionnaire received",
    soonTitle: "Your plan is on the way",
    soonLead: "Your answers are with RYZE. A trainer will now analyse everything and build your training and nutrition plans.",
    soonSubmitted: "Questionnaire received",
    soonTrainer: "Trainer allocation",
    soonBuilding: "Plan in preparation",
    soonNotification: "Notification",
    soonTrainerPending: "A RYZE administrator is choosing the trainer for your service.",
    soonTrainerNamed: (name) => `${name} is now working on your plan.`,
    soonBuildingText: "Your training and nutrition are being built and reviewed before delivery.",
    soonNotificationText: "You will be notified in the app and by email as soon as your plan is ready.",
    viewSummary: "See my answer summary",
    editAnswers: "Edit my answers",
    viewService: "View service status",
    deliveredTitle: "Your plan has been delivered",
    deliveredText: "Your training programme is available in My Programs and your nutrition plan is on your service page.",
    openTraining: "Open my training programme",
    openAcademy: "Open the Premium L2 academy",
    summaryKicker: "Your Premium Level 2",
    summaryTitle: "Your answer summary",
    summaryLead: "A clear view of what you told us and what happens next. The final plan is written and reviewed by a RYZE trainer.",
    analysisTitle: "Initial reading of your answers",
    analysisNote: "This is an automatic summary, not a medical assessment and not your final plan.",
    analysisGoal: "Goal",
    analysisTraining: "Training",
    analysisNutrition: "Nutrition",
    analysisRecovery: "Recovery and lifestyle",
    analysisHealth: "Health and safety",
    analysisNext: "Next step",
    flagMedical: "A medical condition was reported.",
    flagInjury: "An injury or pain was reported.",
    flagMedication: "Medication was reported.",
    flagMovement: "A movement restriction was reported.",
    flagClearance: "Medical clearance was not confirmed.",
    flagSmoking: "Smoking was reported.",
    noHealthFlags: "No health points were flagged in your answers.",
    answersTitle: "Everything you told us",
    answersLead: "This is the same content included in the PDF you can download.",
    showAnswers: "Show all answers",
    receiptTitle: "Purchase receipt",
    receiptReference: "Reference",
    receiptDate: "Date",
    receiptAmount: "Amount",
    receiptMethod: "Payment method",
    receiptStatus: "Status",
    receiptTest: "Test Mode",
    receiptTestText: "This purchase was completed in Test Mode and no money was charged.",
    receiptPending: "Purchase record not found yet.",
    receiptStatusCompleted: "Completed",
    download: "Download answers and receipt (PDF)",
    downloadNote: "In the print window, choose “Save as PDF”.",
    printTitle: "Premium Level 2 — answers and purchase",
    printAnswers: "Questionnaire answers",
    printReceipt: "Purchase record",
    printFooter: "This document contains personal and health information. Keep it private. It is an educational and coaching record, not a medical document.",
    notAnswered: "Not answered",
    yes: "Yes",
    no: "No",
    status: "Status",
    planPending: "Your plan is still being prepared. You will be notified in the app and by email.",
    planPendingText: "You can return to this summary at any time.",
    serviceLink: "Open the full service page",
    purchaseFallback: "Purchase details will appear here when the record is available.",
    emailNote: "The notification is sent to the email on your RYZE account.",
    sessions: (count) => `${count} sessions per week`,
    minutes: (count) => `${count} minutes per session`,
    meals: (count) => `${count} meals per day`,
    sleep: (hours) => `${hours} hours of sleep per night`
  },
  pt: {
    loading: "A carregar a tua experiência Premium Nível 2…",
    errorTitle: "Não foi possível abrir a tua experiência",
    errorText: "Tenta novamente. A tua compra e as tuas respostas estão seguras.",
    retry: "Tentar de novo",
    back: "Premium Nível 2",
    preparingKicker: "Compra confirmada",
    preparingTitle: "Estamos a preparar a tua experiência completa",
    preparingLead: "O teu pagamento está confirmado. A seguir vem o questionário que o teu treinador RYZE vai usar para construir o teu plano individual.",
    preparingSteps: ["Pagamento confirmado", "Acesso ligado à tua conta", "Questionário pronto"],
    preparingDone: "Está tudo pronto",
    preparingButton: "Começar o meu questionário",
    questionnaireKicker: "O teu Premium Nível 2",
    questionnaireTitle: "Conta-nos sobre ti",
    questionnaireLead: "Responde com sinceridade e por completo. O teu treinador usa cada detalhe para construir o treino e a nutrição à volta da tua vida.",
    step: (current, total) => `Secção ${current} de ${total}`,
    previous: "Anterior",
    next: "Seguinte",
    submit: "Enviar as minhas respostas",
    submitting: "A enviar…",
    sectionProgress: (done, total) => `${done} de ${total} secções revistas`,
    required: "Esta resposta é obrigatória.",
    fix: "Completa as respostas assinaladas para continuar.",
    notice: "As tuas respostas ficam neste separador até as enviares. Não feches a página antes de terminar.",
    soonKicker: "Questionário recebido",
    soonTitle: "O teu plano está a caminho",
    soonLead: "As tuas respostas estão com a RYZE. Um treinador vai agora analisar tudo e construir os teus planos de treino e nutrição.",
    soonSubmitted: "Questionário recebido",
    soonTrainer: "Atribuição de treinador",
    soonBuilding: "Plano em preparação",
    soonNotification: "Notificação",
    soonTrainerPending: "Um administrador RYZE está a escolher o treinador para o teu serviço.",
    soonTrainerNamed: (name) => `${name} está agora a trabalhar no teu plano.`,
    soonBuildingText: "O teu treino e a tua nutrição estão a ser construídos e revistos antes da entrega.",
    soonNotificationText: "Vais ser notificado na app e por email assim que o teu plano estiver pronto.",
    viewSummary: "Ver o resumo das minhas respostas",
    editAnswers: "Editar as minhas respostas",
    viewService: "Ver estado do serviço",
    deliveredTitle: "O teu plano foi entregue",
    deliveredText: "O teu programa de treino está em Os meus programas e o plano de nutrição está na página do teu serviço.",
    openTraining: "Abrir o meu programa de treino",
    openAcademy: "Abrir a academia Premium L2",
    summaryKicker: "O teu Premium Nível 2",
    summaryTitle: "Resumo das tuas respostas",
    summaryLead: "Uma visão clara do que nos contaste e do que acontece a seguir. O plano final é escrito e revisto por um treinador RYZE.",
    analysisTitle: "Leitura inicial das tuas respostas",
    analysisNote: "Este é um resumo automático, não é uma avaliação médica nem o teu plano final.",
    analysisGoal: "Objetivo",
    analysisTraining: "Treino",
    analysisNutrition: "Nutrição",
    analysisRecovery: "Recuperação e estilo de vida",
    analysisHealth: "Saúde e segurança",
    analysisNext: "Próximo passo",
    flagMedical: "Foi indicada uma condição médica.",
    flagInjury: "Foi indicada uma lesão ou dor.",
    flagMedication: "Foi indicada medicação.",
    flagMovement: "Foi indicada uma restrição de movimento.",
    flagClearance: "A autorização médica não foi confirmada.",
    flagSmoking: "Foi indicado consumo de tabaco.",
    noHealthFlags: "Não foram assinalados pontos de saúde nas tuas respostas.",
    answersTitle: "Tudo o que nos contaste",
    answersLead: "Este é o mesmo conteúdo incluído no PDF que podes descarregar.",
    showAnswers: "Mostrar todas as respostas",
    receiptTitle: "Comprovativo de compra",
    receiptReference: "Referência",
    receiptDate: "Data",
    receiptAmount: "Valor",
    receiptMethod: "Método de pagamento",
    receiptStatus: "Estado",
    receiptTest: "Modo de Teste",
    receiptTestText: "Esta compra foi concluída em Modo de Teste e não foi cobrado qualquer valor.",
    receiptPending: "O registo da compra ainda não está disponível.",
    receiptStatusCompleted: "Concluída",
    download: "Descarregar respostas e comprovativo (PDF)",
    downloadNote: "Na janela de impressão, escolhe «Guardar como PDF».",
    printTitle: "Premium Nível 2 — respostas e compra",
    printAnswers: "Respostas ao questionário",
    printReceipt: "Registo de compra",
    printFooter: "Este documento contém informação pessoal e de saúde. Mantém-o privado. É um registo educativo e de acompanhamento, não um documento médico.",
    notAnswered: "Sem resposta",
    yes: "Sim",
    no: "Não",
    status: "Estado",
    planPending: "O teu plano ainda está a ser preparado. Vais ser notificado na app e por email.",
    planPendingText: "Podes voltar a este resumo quando quiseres.",
    serviceLink: "Abrir a página completa do serviço",
    purchaseFallback: "Os detalhes da compra aparecem aqui quando o registo estiver disponível.",
    emailNote: "A notificação é enviada para o email da tua conta RYZE.",
    sessions: (count) => `${count} treinos por semana`,
    minutes: (count) => `${count} minutos por treino`,
    meals: (count) => `${count} refeições por dia`,
    sleep: (hours) => `${hours} horas de sono por noite`
  }
};

const mapFormErrors = (error: unknown): Record<string, string> => {
  if (!(error instanceof ApiError) || error.status !== 400) return {};
  const fields: Record<string, string> = {};
  for (const detail of error.details) {
    const separator = detail.indexOf(": ");
    if (separator === -1) continue;
    const key = detail.slice(0, separator);
    const message = detail.slice(separator + 2);
    if (key && message) fields[key] = message;
  }
  return fields;
};

const requestPath = (requestID: string) => `/services/premium-level-2/requests/${requestID}`;
const experiencePath = (requestID: string) => `/services/premium-level-2/experience/${requestID}`;

const latestPurchase = (purchases: PurchaseHistoryEntry[], programID: string): PurchaseHistoryEntry | null =>
  [...purchases]
    .filter((purchase) => purchase.program_id === programID && purchase.status === "completed")
    .sort((left, right) => new Date(right.created_at).getTime() - new Date(left.created_at).getTime())[0] ?? null;

// PremiumLevel2ExperienceEntryPage resolves the request created by a purchase
// and hands the buyer to its experience. It is the destination of the checkout
// button, so the client never has to hunt for the service they just bought.
export const PremiumLevel2ExperienceEntryPage = () => {
  const history = useHistory();
  const [failed, setFailed] = useState(false);
  const requestedPreparing = new URLSearchParams(useLocation().search).has("preparing");

  useEffect(() => {
    let cancelled = false;
    fetchMyPremiumLevel2Requests()
      .then((requests) => {
        if (cancelled) return;
        const activeRequests = requests.filter((request) => request.status !== "cancelled");
        if (activeRequests.length === 0) {
          history.replace("/services/premium-level-2");
          return;
        }
        const newest = [...activeRequests].sort(
          (left, right) => new Date(right.created_at).getTime() - new Date(left.created_at).getTime()
        )[0];
        history.replace(
          `${experiencePath(newest.id)}${requestedPreparing || newest.status === "awaiting_intake" ? "?preparing=1" : ""}`
        );
      })
      .catch((error) => {
        if (cancelled) return;
        if (error instanceof ApiError && error.status === 401) {
          history.replace("/login");
          return;
        }
        setFailed(true);
      });
    return () => {
      cancelled = true;
    };
  }, [history, requestedPreparing]);

  return (
    <PageWrapper className={ui.page}>
      <div className={styles.loadingPage}>
        {failed ? (
          <div className={styles.loadingCard}>
            <strong>{COPY.en.errorTitle}</strong>
            <p>{COPY.en.errorText}</p>
            <Button variant="secondary" size="small" onClick={() => window.location.reload()}>
              {COPY.en.retry}
            </Button>
          </div>
        ) : (
          <div className={styles.loadingCard} role="status" aria-live="polite">
            <Loader2 className={styles.spin} aria-hidden="true" />
            <p>{COPY.en.loading}</p>
          </div>
        )}
      </div>
    </PageWrapper>
  );
};

// PremiumLevel2ExperiencePage is the post-purchase experience: a short
// preparing moment, the full bilingual intake, the honest “plan on the way”
// state, and the final summary with a printable answers/receipt document.
export const PremiumLevel2ExperiencePage = () => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const tr = useGettext(PREMIUM_LEVEL2_PT, QUESTIONNAIRE_PT, COMMERCE_PT);
  const { formatDate, formatMoney } = useI18n();
  const { requestID } = useParams<{ requestID: string }>();
  const history = useHistory();
  const location = useLocation();

  const [catalog, setCatalog] = useState<PremiumLevel2Catalog | null>(null);
  const [detail, setDetail] = useState<PremiumLevel2ClientDetail | null>(null);
  const [purchases, setPurchases] = useState<PurchaseHistoryEntry[]>([]);
  const [answers, setAnswers] = useState<Answers>({});
  const [view, setView] = useState<View>("questionnaire");
  const [section, setSection] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [submitError, setSubmitError] = useState("");
  const [preparingStep, setPreparingStep] = useState(0);
  const [showAllAnswers, setShowAllAnswers] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setLoadError("");
    try {
      const [nextCatalog, nextDetail, nextPurchases] = await Promise.all([
        fetchPremiumLevel2Catalog(),
        fetchMyPremiumLevel2Request(requestID),
        fetchMyPurchases().catch(() => [])
      ]);
      setCatalog(nextCatalog);
      setDetail(nextDetail);
      setPurchases(nextPurchases);
      if (nextDetail.intake) {
        setAnswers(nextDetail.intake.answers as Answers);
      }

      const editable = EDITABLE_STATUSES.has(nextDetail.request.status);
      const hasIntake = Boolean(nextDetail.intake);
      const wantsPreparing = new URLSearchParams(location.search).get("preparing") === "1";
      const wantsQuestionnaire = new URLSearchParams(location.search).get("step") === "questionnaire";

      if (nextDetail.request.status === "delivered") {
        setView("summary");
      } else if (wantsQuestionnaire && editable) {
        setView("questionnaire");
      } else if (wantsPreparing && !hasIntake && nextDetail.request.status === "awaiting_intake") {
        setView("preparing");
      } else if (editable && !hasIntake) {
        setView("questionnaire");
      } else {
        setView("soon");
      }
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      setLoadError(text.errorText);
    } finally {
      setLoading(false);
    }
  }, [history, location.search, requestID, text.errorText]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (view !== "preparing") return;
    const timers = [
      window.setTimeout(() => setPreparingStep(1), 650),
      window.setTimeout(() => setPreparingStep(2), 1450),
      window.setTimeout(() => setPreparingStep(3), 2350),
      window.setTimeout(() => {
        setView("questionnaire");
        history.replace(experiencePath(requestID));
      }, 3800)
    ];
    return () => timers.forEach((timer) => window.clearTimeout(timer));
  }, [history, requestID, view]);

  const sections = catalog?.sections ?? [];
  const current = sections[section];
  const allQuestions = useMemo(
    () => sections.flatMap((entry) => entry.questions),
    [sections]
  );

  const formatAnswer = useCallback(
    (question: PremiumLevel2Question, value: AnswerValue | undefined): string => {
      if (isEmptyPremiumLevel2Answer(value)) return text.notAnswered;
      if (typeof value === "boolean") return value ? text.yes : text.no;
      if (Array.isArray(value)) {
        return value
          .map((entry) => {
            const option = question.options?.find((item) => item.value === entry);
            return option ? tr(option.label) : String(entry);
          })
          .join(", ");
      }
      if (question.options) {
        const option = question.options.find((item) => item.value === value);
        if (option) return tr(option.label);
      }
      return String(value);
    },
    [text.notAnswered, text.no, text.yes, tr]
  );

  const answerOf = useCallback((key: string) => allQuestions.find((question) => question.key === key), [allQuestions]);
  const valueOf = useCallback(
    (key: string) => {
      const question = answerOf(key);
      return question ? formatAnswer(question, answers[key]) : text.notAnswered;
    },
    [answerOf, answers, formatAnswer, text.notAnswered]
  );

  const analysis = useMemo(() => {
    const training = [
      valueOf("sessions_per_week") === text.notAnswered ? "" : text.sessions(Number(answers.sessions_per_week)),
      valueOf("session_minutes") === text.notAnswered ? "" : text.minutes(Number(answers.session_minutes)),
      valueOf("training_location"),
      valueOf("training_experience")
    ].filter(Boolean);
    const nutrition = [
      valueOf("dietary_pattern"),
      valueOf("current_nutrition_quality"),
      valueOf("meals_per_day") === text.notAnswered ? "" : text.meals(Number(answers.meals_per_day)),
      valueOf("allergies") !== text.notAnswered ? `${tr("Allergies")}: ${valueOf("allergies")}` : "",
      valueOf("intolerances") !== text.notAnswered ? `${tr("Intolerances")}: ${valueOf("intolerances")}` : ""
    ].filter(Boolean);
    const recovery = [
      valueOf("activity_level"),
      valueOf("sleep_hours") === text.notAnswered ? "" : text.sleep(String(answers.sleep_hours)),
      valueOf("sleep_quality"),
      valueOf("stress_level")
    ].filter(Boolean);
    const healthFlags = [
      answers.has_medical_conditions === true ? text.flagMedical : "",
      answers.has_injury_history === true ? text.flagInjury : "",
      answers.takes_medication === true ? text.flagMedication : "",
      answers.has_movement_limitations === true ? text.flagMovement : "",
      answers.medical_clearance === false ? text.flagClearance : "",
      answers.smoking === true ? text.flagSmoking : ""
    ].filter(Boolean);
    return [
      {
        key: "goal",
        title: text.analysisGoal,
        text: [valueOf("primary_goal"), valueOf("specific_goal")].filter((entry) => entry !== text.notAnswered).join(" · ")
      },
      { key: "training", title: text.analysisTraining, text: training.join(" · ") },
      { key: "nutrition", title: text.analysisNutrition, text: nutrition.join(" · ") },
      { key: "recovery", title: text.analysisRecovery, text: recovery.join(" · ") },
      {
        key: "health",
        title: text.analysisHealth,
        text: healthFlags.length > 0 ? healthFlags.join(" ") : text.noHealthFlags
      },
      {
        key: "next",
        title: text.analysisNext,
        text: detail?.request.status === "delivered" ? text.deliveredText : text.soonNotificationText
      }
    ];
  }, [answers, detail?.request.status, text, tr, valueOf]);

  const purchase = detail ? latestPurchase(purchases, detail.request.program_id) : null;
  const editable = detail ? EDITABLE_STATUSES.has(detail.request.status) : false;
  const delivered = detail?.request.status === "delivered";

  const validateKeys = useCallback(
    (questions: PremiumLevel2Question[]) => {
      const errors: Record<string, string> = {};
      for (const question of questions) {
        if (question.required && isEmptyPremiumLevel2Answer(answers[question.key])) {
          errors[question.key] = text.required;
        }
      }
      return errors;
    },
    [answers, text.required]
  );

  const goNext = () => {
    if (!current) return;
    const errors = validateKeys(current.questions);
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      setSubmitError(text.fix);
      return;
    }
    setFieldErrors({});
    setSubmitError("");
    if (section < sections.length - 1) {
      setSection(section + 1);
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
  };

  const submit = async () => {
    if (!catalog) return;
    const errors = validateKeys(allQuestions);
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      setSubmitError(text.fix);
      const firstSection = sections.findIndex((entry) => entry.questions.some((question) => errors[question.key]));
      if (firstSection >= 0) setSection(firstSection);
      window.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }
    const payload: Record<string, unknown> = {};
    for (const question of allQuestions) {
      const value = answers[question.key];
      if (isEmptyPremiumLevel2Answer(value)) continue;
      payload[question.key] = typeof value === "string" ? value.trim() : value;
    }
    setSubmitting(true);
    setSubmitError("");
    setFieldErrors({});
    try {
      const result = await submitPremiumLevel2Intake(requestID, payload);
      setDetail((currentDetail) =>
        currentDetail ? { ...currentDetail, request: result.request, intake: result.intake } : currentDetail
      );
      setAnswers(result.intake.answers as Answers);
      setView("soon");
      history.replace(experiencePath(requestID));
      window.scrollTo({ top: 0, behavior: "smooth" });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      if (error instanceof ApiError && error.code === "PREMIUM_LEVEL_2_NOT_EDITABLE") {
        await load();
        setView("soon");
        return;
      }
      const fields = mapFormErrors(error);
      if (Object.keys(fields).length > 0) {
        setFieldErrors(fields);
        const firstSection = sections.findIndex((entry) => entry.questions.some((question) => fields[question.key]));
        if (firstSection >= 0) setSection(firstSection);
      }
      setSubmitError(
        error instanceof ApiError && error.code === "VALIDATION_ERROR"
          ? text.fix
          : text.errorText
      );
    } finally {
      setSubmitting(false);
    }
  };

  const startQuestionnaire = () => {
    setView("questionnaire");
    history.replace(experiencePath(requestID));
  };

  const printDocument = () => window.print();

  if (loading) {
    return (
      <PageWrapper className={ui.page}>
        <div className={styles.loadingPage}>
          <div className={styles.loadingCard} role="status" aria-live="polite">
            <Loader2 className={styles.spin} aria-hidden="true" />
            <p>{text.loading}</p>
          </div>
        </div>
      </PageWrapper>
    );
  }

  if (loadError || !detail || !catalog) {
    return (
      <PageWrapper className={ui.page}>
        <div className={styles.loadingPage}>
          <div className={styles.loadingCard}>
            <strong>{text.errorTitle}</strong>
            <p>{loadError || text.errorText}</p>
            <Button variant="secondary" size="small" icon={<RefreshCw size={15} />} onClick={() => void load()}>
              {text.retry}
            </Button>
          </div>
        </div>
      </PageWrapper>
    );
  }

  const request = detail.request;

  return (
    <PageWrapper className={ui.page}>
      <div className={styles.experiencePage}>
        <Container size="wide" className={styles.container}>
          <div className={styles.topRow}>
            <Link className={styles.back} to="/services/premium-level-2">
              <ArrowLeft aria-hidden="true" /> {text.back}
            </Link>
            <span className={styles.statusPill}>
              <ShieldCheck aria-hidden="true" />
              {tr("Premium Level 2")} · {request.offer_name}
            </span>
          </div>

          {view === "preparing" ? (
            <section className={styles.preparing} aria-labelledby="preparing-title">
              <div className={styles.orb} aria-hidden="true">
                <span />
                <span />
                <span />
                <CheckCircle2 />
              </div>
              <p className={styles.kicker}>
                <Sparkles aria-hidden="true" /> {text.preparingKicker}
              </p>
              <h1 id="preparing-title" className={styles.title}>
                {text.preparingTitle}
              </h1>
              <p className={styles.lead}>{text.preparingLead}</p>
              <ol className={styles.preparingSteps}>
                {text.preparingSteps.map((step, index) => (
                  <li key={step} data-done={preparingStep > index}>
                    <span className={styles.stepIcon} aria-hidden="true">
                      {preparingStep > index ? <Check /> : <Clock3 />}
                    </span>
                    {step}
                  </li>
                ))}
              </ol>
              <button type="button" className={styles.primary} onClick={startQuestionnaire}>
                {preparingStep >= text.preparingSteps.length ? text.preparingButton : text.preparingDone}
                <ArrowRight aria-hidden="true" />
              </button>
            </section>
          ) : null}

          {view === "questionnaire" ? (
            <section aria-labelledby="questionnaire-title">
              <div className={styles.heading}>
                <div>
                  <p className={styles.kicker}>
                    <ClipboardList aria-hidden="true" /> {text.questionnaireKicker}
                  </p>
                  <h1 id="questionnaire-title" className={styles.title}>
                    {text.questionnaireTitle}
                  </h1>
                  <p className={styles.lead}>{text.questionnaireLead}</p>
                </div>
                <div className={styles.progressCard}>
                  <strong>{text.step(section + 1, sections.length)}</strong>
                  <span>{current ? tr(current.title) : ""}</span>
                  <div className={styles.progressBar} aria-hidden="true">
                    <span style={{ width: `${((section + 1) / sections.length) * 100}%` }} />
                  </div>
                </div>
              </div>

              <nav className={styles.sectionNav} aria-label={text.questionnaireTitle}>
                {sections.map((entry, index) => (
                  <button
                    key={entry.key}
                    type="button"
                    className={styles.sectionChip}
                    data-current={index === section}
                    onClick={() => {
                      setSection(index);
                      setFieldErrors({});
                      setSubmitError("");
                      window.scrollTo({ top: 0, behavior: "smooth" });
                    }}
                  >
                    {index + 1}. {tr(entry.title)}
                  </button>
                ))}
              </nav>

              <div className={styles.questionnaireGrid}>
                <div className={styles.sectionCard}>
                  {current ? (
                    <>
                      <div className={styles.sectionHeading}>
                        <span>{String(section + 1).padStart(2, "0")}</span>
                        <div>
                          <h2>{tr(current.title)}</h2>
                          <p>{tr(current.description)}</p>
                        </div>
                      </div>
                      <div className={styles.fields}>
                        {current.questions.map((question) => (
                          <QuestionField
                            key={question.key}
                            question={question}
                            value={answers[question.key]}
                            error={fieldErrors[question.key] ?? ""}
                            disabled={submitting}
                            onChange={(value) => {
                              setAnswers((state) => ({ ...state, [question.key]: value }));
                              setFieldErrors((state) => {
                                if (!(question.key in state)) return state;
                                const next = { ...state };
                                delete next[question.key];
                                return next;
                              });
                              setSubmitError("");
                            }}
                          />
                        ))}
                      </div>
                    </>
                  ) : null}
                  {submitError ? (
                    <p className={styles.errorText} role="alert">
                      {submitError}
                    </p>
                  ) : null}
                  <div className={styles.questionActions}>
                    <button
                      type="button"
                      className={styles.secondary}
                      disabled={section === 0 || submitting}
                      onClick={() => {
                        setSection(Math.max(0, section - 1));
                        setFieldErrors({});
                        setSubmitError("");
                        window.scrollTo({ top: 0, behavior: "smooth" });
                      }}
                    >
                      <ArrowLeft aria-hidden="true" /> {text.previous}
                    </button>
                    {section < sections.length - 1 ? (
                      <button type="button" className={styles.primary} onClick={goNext}>
                        {text.next} <ArrowRight aria-hidden="true" />
                      </button>
                    ) : (
                      <button type="button" className={styles.primary} disabled={submitting} onClick={() => void submit()}>
                        {submitting ? <Loader2 className={styles.spin} aria-hidden="true" /> : <CheckCircle2 aria-hidden="true" />}
                        {submitting ? text.submitting : text.submit}
                      </button>
                    )}
                  </div>
                </div>

                <aside className={styles.side}>
                  <div className={styles.sideCard}>
                    <strong>{text.notice}</strong>
                    <p>{text.emailNote}</p>
                  </div>
                  <div className={styles.sideCard}>
                    <span>{text.sectionProgress(section, sections.length)}</span>
                    <div className={styles.progressBar} aria-hidden="true">
                      <span style={{ width: `${((section + 1) / sections.length) * 100}%` }} />
                    </div>
                  </div>
                </aside>
              </div>
            </section>
          ) : null}

          {view === "soon" ? (
            <section aria-labelledby="soon-title">
              <div className={styles.soonHero}>
                <div>
                  <p className={styles.kicker}>
                    <Bell aria-hidden="true" /> {text.soonKicker}
                  </p>
                  <h1 id="soon-title" className={styles.title}>
                    {text.soonTitle}
                  </h1>
                  <p className={styles.lead}>{text.soonLead}</p>
                </div>
                <div className={styles.bellOrb} aria-hidden="true">
                  <Bell />
                </div>
              </div>

              <div className={styles.timeline}>
                <div className={styles.timelineItem} data-done="true">
                  <span className={styles.stepIcon}><Check /></span>
                  <div>
                    <strong>{text.soonSubmitted}</strong>
                    <p>{request.intake_version > 0 ? tr("Revision {version} saved.", { version: request.intake_version }) : ""}</p>
                  </div>
                </div>
                <div className={styles.timelineItem} data-done={request.trainer_name ? "true" : "false"}>
                  <span className={styles.stepIcon}>{request.trainer_name ? <Check /> : <Clock3 />}</span>
                  <div>
                    <strong>{text.soonTrainer}</strong>
                    <p>{request.trainer_name ? text.soonTrainerNamed(request.trainer_name) : text.soonTrainerPending}</p>
                  </div>
                </div>
                <div
                  className={styles.timelineItem}
                  data-done={["in_progress", "ready_for_delivery", "delivered"].includes(request.status) ? "true" : "false"}
                >
                  <span className={styles.stepIcon}>
                    {["in_progress", "ready_for_delivery", "delivered"].includes(request.status) ? <Check /> : <Dumbbell />}
                  </span>
                  <div>
                    <strong>{text.soonBuilding}</strong>
                    <p>{text.soonBuildingText}</p>
                  </div>
                </div>
                <div className={styles.timelineItem} data-done="false">
                  <span className={styles.stepIcon}><Mail /></span>
                  <div>
                    <strong>{text.soonNotification}</strong>
                    <p>{text.soonNotificationText}</p>
                  </div>
                </div>
              </div>

              {delivered ? (
                <div className={styles.deliveredCard}>
                  <CheckCircle2 aria-hidden="true" />
                  <div>
                    <h2>{text.deliveredTitle}</h2>
                    <p>{text.deliveredText}</p>
                    <div className={styles.inlineActions}>
                      {request.training_program_id ? (
                        <Link className={styles.primary} to={`/services/my-programs/${request.training_program_id}`}>
                          {text.openTraining} <ArrowRight aria-hidden="true" />
                        </Link>
                      ) : null}
                      <Link className={styles.secondary} to="/services/premium-level-2/academy">
                        <BookOpen aria-hidden="true" /> {text.openAcademy}
                      </Link>
                    </div>
                  </div>
                </div>
              ) : null}

              <div className={styles.inlineActions}>
                <button type="button" className={styles.primary} onClick={() => setView("summary")}>
                  {text.viewSummary} <ArrowRight aria-hidden="true" />
                </button>
                {editable ? (
                  <button
                    type="button"
                    className={styles.secondary}
                    onClick={() => {
                      setView("questionnaire");
                      setSection(0);
                      history.replace(`${experiencePath(requestID)}?step=questionnaire`);
                    }}
                  >
                    <ClipboardList aria-hidden="true" /> {text.editAnswers}
                  </button>
                ) : null}
                <Link className={styles.secondary} to={requestPath(requestID)}>
                  {text.viewService}
                </Link>
              </div>
            </section>
          ) : null}

          {view === "summary" ? (
            <section aria-labelledby="summary-title">
              <div className={styles.heading}>
                <div>
                  <p className={styles.kicker}>
                    <FileText aria-hidden="true" /> {text.summaryKicker}
                  </p>
                  <h1 id="summary-title" className={styles.title}>
                    {text.summaryTitle}
                  </h1>
                  <p className={styles.lead}>{text.summaryLead}</p>
                </div>
                <div className={styles.downloadCard}>
                  <strong>{text.download}</strong>
                  <span>{text.downloadNote}</span>
                  <button type="button" className={styles.primary} onClick={printDocument}>
                    <Download aria-hidden="true" /> {text.download}
                  </button>
                </div>
              </div>

              <div className={styles.analysisGrid}>
                <div className={styles.analysisHead}>
                  <h2>{text.analysisTitle}</h2>
                  <p>{text.analysisNote}</p>
                </div>
                {analysis.map((item) => (
                  <article key={item.key} className={styles.analysisCard}>
                    <span>{item.title}</span>
                    <p>{item.text || text.notAnswered}</p>
                  </article>
                ))}
              </div>

              {delivered ? (
                <div className={styles.deliveredCard}>
                  <CheckCircle2 aria-hidden="true" />
                  <div>
                    <h2>{text.deliveredTitle}</h2>
                    <p>{text.deliveredText}</p>
                    <div className={styles.inlineActions}>
                      {request.training_program_id ? (
                        <Link className={styles.primary} to={`/services/my-programs/${request.training_program_id}`}>
                          <Dumbbell aria-hidden="true" /> {text.openTraining}
                        </Link>
                      ) : null}
                      <Link className={styles.secondary} to="/services/premium-level-2/academy">
                        <BookOpen aria-hidden="true" /> {text.openAcademy}
                      </Link>
                    </div>
                  </div>
                </div>
              ) : (
                <div className={styles.pendingCard}>
                  <Clock3 aria-hidden="true" />
                  <div>
                    <h2>{text.soonBuilding}</h2>
                    <p>{text.planPending}</p>
                  </div>
                </div>
              )}

              <div className={styles.summaryGrid}>
                <section className={styles.receiptCard}>
                  <div className={styles.sectionHeading}>
                    <span>€</span>
                    <div>
                      <h2>{text.receiptTitle}</h2>
                      <p>{request.offer_name}</p>
                    </div>
                  </div>
                  {purchase ? (
                    <dl className={styles.receiptList}>
                      <div>
                        <dt>{text.receiptReference}</dt>
                        <dd>{purchase.id.slice(0, 8).toUpperCase()}</dd>
                      </div>
                      <div>
                        <dt>{text.receiptDate}</dt>
                        <dd>{formatDate(purchase.created_at, { year: "numeric", month: "long", day: "numeric" })}</dd>
                      </div>
                      <div>
                        <dt>{text.receiptAmount}</dt>
                        <dd>{formatMoney(purchase.price_minor_units, purchase.currency || "EUR")}</dd>
                      </div>
                      <div>
                        <dt>{text.receiptMethod}</dt>
                        <dd>{purchase.test ? text.receiptTest : (purchase.payment_method ?? "—")}</dd>
                      </div>
                      <div>
                        <dt>{text.receiptStatus}</dt>
                        <dd>{text.receiptStatusCompleted}</dd>
                      </div>
                    </dl>
                  ) : (
                    <p>{text.receiptPending}</p>
                  )}
                  {purchase?.test ? <p className={styles.note}>{text.receiptTestText}</p> : null}
                </section>

                <section className={styles.answersCard}>
                  <div className={styles.sectionHeading}>
                    <span><ClipboardList /></span>
                    <div>
                      <h2>{text.answersTitle}</h2>
                      <p>{text.answersLead}</p>
                    </div>
                  </div>
                  <button type="button" className={styles.secondary} onClick={() => setShowAllAnswers((value) => !value)}>
                    {text.showAnswers}
                  </button>
                  {showAllAnswers ? (
                    <div className={styles.allAnswers}>
                      {sections.map((entry) => (
                        <div key={entry.key}>
                          <h3>{tr(entry.title)}</h3>
                          <dl>
                            {entry.questions.map((question) => (
                              <div key={question.key}>
                                <dt>{tr(question.label)}</dt>
                                <dd>{formatAnswer(question, answers[question.key])}</dd>
                              </div>
                            ))}
                          </dl>
                        </div>
                      ))}
                    </div>
                  ) : null}
                </section>
              </div>

              <div className={styles.inlineActions}>
                {editable ? (
                  <button
                    type="button"
                    className={styles.secondary}
                    onClick={() => {
                      setView("questionnaire");
                      setSection(0);
                      history.replace(`${experiencePath(requestID)}?step=questionnaire`);
                    }}
                  >
                    {text.editAnswers}
                  </button>
                ) : null}
                <Link className={styles.secondary} to={requestPath(requestID)}>
                  {text.serviceLink}
                </Link>
                <button type="button" className={styles.ghost} onClick={() => setView("soon")}>
                  {text.planPendingText}
                </button>
              </div>
            </section>
          ) : null}
        </Container>
      </div>

      <article className={styles.printOnly} aria-hidden="true">
        <header>
          <h1>{text.printTitle}</h1>
          <p>{request.offer_name}</p>
          <p>
            {formatDate(request.created_at, { year: "numeric", month: "long", day: "numeric" })} ·{" "}
            {request.trainer_name || tr("Allocation pending")}
          </p>
        </header>
        <section>
          <h2>{text.printAnswers}</h2>
          {sections.map((entry) => (
            <div key={entry.key}>
              <h3>{tr(entry.title)}</h3>
              <dl>
                {entry.questions.map((question) => (
                  <div key={question.key}>
                    <dt>{tr(question.label)}</dt>
                    <dd>{formatAnswer(question, answers[question.key])}</dd>
                  </div>
                ))}
              </dl>
            </div>
          ))}
        </section>
        <section>
          <h2>{text.printReceipt}</h2>
          {purchase ? (
            <dl>
              <div><dt>{text.receiptReference}</dt><dd>{purchase.id}</dd></div>
              <div><dt>{text.receiptDate}</dt><dd>{formatDate(purchase.created_at)}</dd></div>
              <div><dt>{text.receiptAmount}</dt><dd>{formatMoney(purchase.price_minor_units, purchase.currency || "EUR")}</dd></div>
              <div><dt>{text.receiptMethod}</dt><dd>{purchase.test ? text.receiptTest : (purchase.payment_method ?? "—")}</dd></div>
              <div><dt>{text.receiptStatus}</dt><dd>{text.receiptStatusCompleted}</dd></div>
            </dl>
          ) : (
            <p>{text.purchaseFallback}</p>
          )}
        </section>
        <footer>{text.printFooter}</footer>
      </article>
    </PageWrapper>
  );
};
