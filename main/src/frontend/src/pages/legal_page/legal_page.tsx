import { Cookie, FileText, Scale, ShieldCheck } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Link, Redirect, useParams } from "react-router-dom";

import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import ui from "@/components/public_ui/public_ui.module.css";
import { LEGAL_ENTITY } from "@/constants/legal";
import { useI18n } from "@/i18n/i18n";
import { openCookieSettings } from "@/services/cookie_consent";
import { joinClassNames } from "@utils/class_names";

import { LEGAL_DOCS, LEGAL_ORDER, type LegalDocId } from "./legal_content";
import styles from "./legal_page.module.css";

const ICONS: Record<LegalDocId, LucideIcon> = { termos: FileText, privacidade: ShieldCheck, cookies: Cookie, "aviso-legal": Scale };

const COPY = {
  en: { label: "Legal", updated: "Last updated", version: "Version", contents: "On this page", documents: "Documents", cookieSettings: "Cookie settings" },
  pt: { label: "Legal", updated: "Última atualização", version: "Versão", contents: "Nesta página", documents: "Documentos", cookieSettings: "Definições de cookies" }
} as const;

// LegalPage shows one of the legal documents (owner request, 2026-11-23:
// Terms, Privacy, Cookies and Legal notice in line with the GDPR and EU
// consumer law), with the other documents and the sections beside it.
export const LegalPage = () => {
  const { doc } = useParams<{ doc?: string }>();
  const { locale, formatDate } = useI18n();
  const text = COPY[locale];
  if (!doc || !LEGAL_ORDER.includes(doc as LegalDocId)) return <Redirect to="/legal/termos" />;
  const id = doc as LegalDocId;
  const document = LEGAL_DOCS[locale][id];
  const Icon = ICONS[id];

  return (
    <PageWrapper className={ui.page}>
      <section className={joinClassNames(ui.toneNavy, styles.hero)}>
        <div className={joinClassNames(ui.container, styles.heroInner)}>
          <p className={ui.label}>{text.label}</p>
          <h1 className={joinClassNames(ui.display, styles.title)}>
            <span className={styles.titleIcon} aria-hidden="true">
              <Icon />
            </span>
            {document.title}
          </h1>
          <p className={ui.lead}>{document.summary}</p>
          <p className={styles.meta}>
            {text.updated}: {formatDate(`${LEGAL_ENTITY.updated}T00:00:00`, { day: "numeric", month: "long", year: "numeric" })} · {text.version}{" "}
            {LEGAL_ENTITY.version}
          </p>
        </div>
      </section>

      <section className={joinClassNames(ui.toneNavy, styles.body)}>
        <div className={joinClassNames(ui.container, styles.layout)}>
          <aside className={styles.aside}>
            <nav className={styles.docs} aria-label={text.documents}>
              {LEGAL_ORDER.map((other) => {
                const OtherIcon = ICONS[other];
                return (
                  <Link key={other} to={`/legal/${other}`} className={joinClassNames(styles.docLink, other === id && styles.docLinkActive)} aria-current={other === id ? "page" : undefined}>
                    <OtherIcon aria-hidden="true" />
                    {LEGAL_DOCS[locale][other].title}
                  </Link>
                );
              })}
            </nav>
            <nav className={styles.toc} aria-label={text.contents}>
              <p>{text.contents}</p>
              <ol>
                {document.sections.map((section) => (
                  <li key={section.id}>
                    <a href={`#${section.id}`}>{section.title}</a>
                  </li>
                ))}
              </ol>
            </nav>
          </aside>

          <article className={joinClassNames(ui.surface, styles.article)}>
            {id === "cookies" ? (
              <button type="button" className={styles.settings} onClick={openCookieSettings}>
                <Cookie aria-hidden="true" /> {text.cookieSettings}
              </button>
            ) : null}
            {document.sections.map((section) => (
              <section key={section.id} id={section.id} className={styles.section}>
                <h2>{section.title}</h2>
                {section.paragraphs?.map((paragraph) => <p key={paragraph}>{paragraph}</p>)}
                {section.list ? (
                  <ul>
                    {section.list.map((item) => (
                      <li key={item}>{item}</li>
                    ))}
                  </ul>
                ) : null}
                {section.table ? (
                  <div className={styles.tableWrap}>
                    <table>
                      <thead>
                        <tr>
                          {section.table.headers.map((header) => (
                            <th key={header} scope="col">
                              {header}
                            </th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {section.table.rows.map((row) => (
                          <tr key={row[0]}>
                            {row.map((cell, index) => (index === 0 ? <th key={index} scope="row">{cell}</th> : <td key={index}>{cell}</td>))}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : null}
              </section>
            ))}
          </article>
        </div>
      </section>
    </PageWrapper>
  );
};
