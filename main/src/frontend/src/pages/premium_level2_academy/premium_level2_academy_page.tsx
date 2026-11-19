import {
  ArrowLeft,
  ArrowRight,
  BookOpen,
  CheckCircle2,
  ChevronRight,
  Dumbbell,
  GraduationCap,
  Leaf,
  Loader2,
  LockKeyhole,
  ShieldCheck,
  Sparkles
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useHistory } from "react-router-dom";

import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { type Copy, useI18n } from "@/i18n/i18n";
import { fetchMyPremiumLevel2Requests, type PremiumLevel2Request } from "@/services/premium_level_2_api";
import { ApiError } from "@utils/http_client";
import { joinClassNames } from "@utils/class_names";

import {
  ACADEMY_CHAPTERS,
  type AcademyBlock,
  type AcademyChapter,
  academyText
} from "./premium_level2_academy_content";
import styles from "./premium_level2_academy_page.module.css";

type PageState = "loading" | "ready" | "restricted" | "error";

const COPY: Copy<{
  kicker: string;
  title: string;
  lead: string;
  included: string;
  deliveredOn: (date: string) => string;
  guide: string;
  chapters: (count: number) => string;
  contents: string;
  weeks: string;
  readingNote: string;
  backExperience: string;
  viewService: string;
  restrictedKicker: string;
  restrictedTitle: string;
  restrictedText: string;
  restrictedAction: string;
  errorTitle: string;
  errorText: string;
  retry: string;
  loading: string;
  references: string;
  previous: string;
  next: string;
  done: string;
}> = {
  en: {
    kicker: "Premium Level 2 academy",
    title: "The method behind your plan",
    lead: "The complete practical guide to hypertrophy, load progression and nutrition — included with your delivered Premium Level 2 service.",
    included: "Included with your delivered Premium L2",
    deliveredOn: (date) => `Delivered ${date}`,
    guide: "Official 8-week guide",
    chapters: (count) => `${count} chapters`,
    contents: "Contents",
    weeks: "WEEKS",
    readingNote: "The guide teaches the method. Your individual training and nutrition are written by your RYZE trainer from your questionnaire.",
    backExperience: "My Premium L2 experience",
    viewService: "Service details",
    restrictedKicker: "Premium Level 2",
    restrictedTitle: "The academy opens with your plan",
    restrictedText: "This area is exclusive to clients whose Premium Level 2 service has been delivered. Once your trainer finishes your plan, the full academy becomes available here.",
    restrictedAction: "See Premium Level 2",
    errorTitle: "We could not open the academy",
    errorText: "Please try again. Your service and plan are safe.",
    retry: "Try again",
    loading: "Opening the academy…",
    references: "References",
    previous: "Previous chapter",
    next: "Next chapter",
    done: "You reached the end"
  },
  pt: {
    kicker: "Academia Premium Nível 2",
    title: "O método por trás do teu plano",
    lead: "O guia prático completo de hipertrofia, progressão de carga e nutrição — incluído no teu serviço Premium Nível 2 entregue.",
    included: "Incluído no teu Premium N2 entregue",
    deliveredOn: (date) => `Entregue a ${date}`,
    guide: "Guia oficial de 8 semanas",
    chapters: (count) => `${count} capítulos`,
    contents: "Conteúdo",
    weeks: "SEMANAS",
    readingNote: "O guia ensina o método. O teu treino e a tua nutrição individuais são escritos pelo teu treinador RYZE a partir do teu questionário.",
    backExperience: "A minha experiência Premium N2",
    viewService: "Detalhes do serviço",
    restrictedKicker: "Premium Nível 2",
    restrictedTitle: "A academia abre com o teu plano",
    restrictedText: "Esta área é exclusiva para clientes cujo serviço Premium Nível 2 já foi entregue. Quando o teu treinador terminar o plano, a academia completa fica disponível aqui.",
    restrictedAction: "Ver Premium Nível 2",
    errorTitle: "Não foi possível abrir a academia",
    errorText: "Tenta novamente. O teu serviço e o teu plano estão seguros.",
    retry: "Tentar de novo",
    loading: "A abrir a academia…",
    references: "Referências",
    previous: "Capítulo anterior",
    next: "Capítulo seguinte",
    done: "Chegaste ao fim"
  }
};

const CHAPTER_ICONS = [BookOpen, Dumbbell, Sparkles, ChevronRight, Leaf, ShieldCheck, CheckCircle2, LockKeyhole, GraduationCap, BookOpen, Sparkles, CheckCircle2];

const renderText = (text: { en: string; pt: string }, locale: "en" | "pt") => academyText(text, locale);

const AcademyBlockView = ({
  block,
  locale,
  referencesLabel
}: {
  block: AcademyBlock;
  locale: "en" | "pt";
  referencesLabel: string;
}) => {
  if (block.kind === "paragraph") {
    return <p className={styles.paragraph}>{renderText(block.text, locale)}</p>;
  }

  if (block.kind === "list") {
    const List = block.ordered ? "ol" : "ul";
    return (
      <List className={joinClassNames(styles.list, block.ordered && styles.orderedList)}>
        {block.items.map((item, index) => (
          <li key={`${index}-${renderText(item, "en").slice(0, 24)}`}>{renderText(item, locale)}</li>
        ))}
      </List>
    );
  }

  if (block.kind === "cards") {
    return (
      <div className={styles.cards}>
        {block.items.map((item) => (
          <article key={renderText(item.title, "en")} className={styles.card}>
            <h3>{renderText(item.title, locale)}</h3>
            <p>{renderText(item.text, locale)}</p>
          </article>
        ))}
      </div>
    );
  }

  if (block.kind === "table") {
    return (
      <div className={styles.tableWrap}>
        <table className={styles.table}>
          <thead>
            <tr>
              {block.headers.map((header) => (
                <th key={renderText(header, "en")}>{renderText(header, locale)}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {block.rows.map((row, rowIndex) => (
              <tr key={rowIndex}>
                {row.map((cell, cellIndex) => (
                  <td key={cellIndex}>{renderText(cell, locale)}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  if (block.kind === "callout") {
    return (
      <aside className={joinClassNames(styles.callout, block.tone === "warning" && styles.warning, block.tone === "success" && styles.success)}>
        {block.title ? <strong>{renderText(block.title, locale)}</strong> : null}
        <p>{renderText(block.text, locale)}</p>
      </aside>
    );
  }

  return (
    <section className={styles.references} aria-label={referencesLabel}>
      <h3>{referencesLabel}</h3>
      <ol>
        {block.items.map((item) => (
          <li key={item.slice(0, 60)}>{item}</li>
        ))}
      </ol>
    </section>
  );
};

// PremiumLevel2AcademyPage is the exclusive education area of the delivered
// Premium Level 2 service. Access is checked against the member's own request
// list; the page never trusts a client-provided plan or entitlement.
export const PremiumLevel2AcademyPage = () => {
  const { locale, formatDate } = useI18n();
  const text = COPY[locale];
  const history = useHistory();
  const [state, setState] = useState<PageState>("loading");
  const [delivered, setDelivered] = useState<PremiumLevel2Request | null>(null);
  const [activeChapter, setActiveChapter] = useState(ACADEMY_CHAPTERS[0].id);

  const load = useCallback(async () => {
    setState("loading");
    try {
      const requests = await fetchMyPremiumLevel2Requests();
      const completed = [...requests]
        .filter((request) => request.status === "delivered")
        .sort((left, right) => new Date(right.delivered_at ?? right.updated_at).getTime() - new Date(left.delivered_at ?? left.updated_at).getTime())[0];
      if (!completed) {
        setDelivered(null);
        setState("restricted");
        return;
      }
      setDelivered(completed);
      setState("ready");
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      setState("error");
    }
  }, [history]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (state !== "ready") return;
    const sections = ACADEMY_CHAPTERS
      .map((chapter) => document.getElementById(`academy-${chapter.id}`))
      .filter((element): element is HTMLElement => Boolean(element));
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries.filter((entry) => entry.isIntersecting).sort((left, right) => left.boundingClientRect.top - right.boundingClientRect.top);
        if (visible[0]) setActiveChapter(visible[0].target.id.replace("academy-", ""));
      },
      { rootMargin: "-18% 0px -68% 0px", threshold: 0 }
    );
    sections.forEach((section) => observer.observe(section));
    return () => observer.disconnect();
  }, [state]);

  const activeIndex = useMemo(
    () => Math.max(0, ACADEMY_CHAPTERS.findIndex((chapter) => chapter.id === activeChapter)),
    [activeChapter]
  );
  if (state === "loading") {
    return (
      <PageWrapper className={styles.page}>
        <div className={styles.stateCard} role="status" aria-live="polite">
          <Loader2 className={styles.spin} aria-hidden="true" />
          <p>{text.loading}</p>
        </div>
      </PageWrapper>
    );
  }

  if (state === "error") {
    return (
      <PageWrapper className={styles.page}>
        <div className={styles.stateCard}>
          <strong>{text.errorTitle}</strong>
          <p>{text.errorText}</p>
          <button type="button" className={styles.primary} onClick={() => void load()}>
            {text.retry}
          </button>
        </div>
      </PageWrapper>
    );
  }

  if (state === "restricted") {
    return (
      <PageWrapper className={styles.page}>
        <div className={styles.restricted}>
          <span className={styles.lock} aria-hidden="true"><LockKeyhole /></span>
          <p className={styles.kicker}>{text.restrictedKicker}</p>
          <h1>{text.restrictedTitle}</h1>
          <p>{text.restrictedText}</p>
          <Link className={styles.primary} to="/services/premium-level-2">
            {text.restrictedAction} <ArrowRight aria-hidden="true" />
          </Link>
        </div>
      </PageWrapper>
    );
  }

  return (
    <PageWrapper className={styles.page}>
      <div className={styles.hero}>
        <div className={styles.heroCopy}>
          <p className={styles.kicker}>
            <GraduationCap aria-hidden="true" /> {text.kicker}
          </p>
          <h1>
            {text.title}
            <span>.</span>
          </h1>
          <p className={styles.lead}>{text.lead}</p>
          <div className={styles.metaRow}>
            <span><ShieldCheck aria-hidden="true" /> {text.included}</span>
            {delivered?.delivered_at ? <span>{text.deliveredOn(formatDate(delivered.delivered_at))}</span> : null}
            <span>{text.guide}</span>
            <span>{text.chapters(ACADEMY_CHAPTERS.length)}</span>
          </div>
          <div className={styles.heroActions}>
            {delivered ? (
              <Link className={styles.secondary} to={`/services/premium-level-2/experience/${delivered.id}`}>
                <ArrowLeft aria-hidden="true" /> {text.backExperience}
              </Link>
            ) : null}
            {delivered ? (
              <Link className={styles.ghost} to={`/services/premium-level-2/requests/${delivered.id}`}>
                {text.viewService} <ArrowRight aria-hidden="true" />
              </Link>
            ) : null}
          </div>
        </div>
        <div className={styles.heroMark} aria-hidden="true">
          <BookOpen />
          <span>8</span>
          <small>{text.weeks}</small>
        </div>
      </div>

      <p className={styles.readingNote}>{text.readingNote}</p>

      <div className={styles.readingProgress} aria-hidden="true">
        <span style={{ width: `${((activeIndex + 1) / ACADEMY_CHAPTERS.length) * 100}%` }} />
      </div>

      <div className={styles.layout}>
        <aside className={styles.toc}>
          <p className={styles.tocTitle}>{text.contents}</p>
          <nav aria-label={text.contents}>
            {ACADEMY_CHAPTERS.map((chapter) => (
              <a
                key={chapter.id}
                href={`#academy-${chapter.id}`}
                className={styles.tocLink}
                data-active={chapter.id === activeChapter}
                onClick={() => setActiveChapter(chapter.id)}
              >
                <span>{chapter.number}</span>
                {renderText(chapter.title, locale)}
              </a>
            ))}
          </nav>
        </aside>

        <div className={styles.content}>
          {ACADEMY_CHAPTERS.map((chapter, index) => (
            <AcademyChapterView
              key={chapter.id}
              chapter={chapter}
              index={index}
              locale={locale}
              referencesLabel={text.references}
            />
          ))}

          <div className={styles.chapterNav}>
            {activeIndex > 0 ? (
              <a className={styles.secondary} href={`#academy-${ACADEMY_CHAPTERS[activeIndex - 1].id}`}>
                <ArrowLeft aria-hidden="true" /> {text.previous}
              </a>
            ) : <span />}
            {activeIndex < ACADEMY_CHAPTERS.length - 1 ? (
              <a className={styles.primary} href={`#academy-${ACADEMY_CHAPTERS[activeIndex + 1].id}`}>
                {text.next} <ArrowRight aria-hidden="true" />
              </a>
            ) : (
              <span className={styles.done}>{text.done}</span>
            )}
          </div>
        </div>
      </div>
    </PageWrapper>
  );
};

const AcademyChapterView = ({
  chapter,
  index,
  locale,
  referencesLabel
}: {
  chapter: AcademyChapter;
  index: number;
  locale: "en" | "pt";
  referencesLabel: string;
}) => {
  const Icon = CHAPTER_ICONS[index % CHAPTER_ICONS.length];
  return (
    <section id={`academy-${chapter.id}`} className={styles.chapter} aria-labelledby={`academy-title-${chapter.id}`}>
      <header className={styles.chapterHeader}>
        <span className={styles.chapterNumber}>{chapter.number}</span>
        <div>
          <p className={styles.chapterKicker}>
            <Icon aria-hidden="true" /> {chapter.number}
          </p>
          <h2 id={`academy-title-${chapter.id}`}>{renderText(chapter.title, locale)}</h2>
          <p>{renderText(chapter.intro, locale)}</p>
        </div>
      </header>
      <div className={styles.chapterBody}>
        {chapter.blocks.map((block, blockIndex) => (
          <AcademyBlockView
            key={`${chapter.id}-${block.kind}-${blockIndex}`}
            block={block}
            locale={locale}
            referencesLabel={referencesLabel}
          />
        ))}
      </div>
    </section>
  );
};
