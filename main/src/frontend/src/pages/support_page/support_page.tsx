import { ArrowLeft, LifeBuoy, MessageSquarePlus } from "lucide-react";
import type { FormEvent } from "react";
import { useCallback, useEffect, useId, useMemo, useRef, useState } from "react";
import { Link, useHistory, useLocation } from "react-router-dom";

import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { SupportThread, type SupportThreadLabels } from "@/components/support_chat/support_thread";
import { useSupportThread, type SupportThreadApi } from "@/components/support_chat/use_support_thread";
import { useI18n } from "@/i18n/i18n";
import {
  fetchMyConversations,
  fetchMyThread,
  fetchSupportEligibility,
  markMyThreadRead,
  newClientMessageId,
  openConversation,
  openSupportStream,
  sendMyMessage,
  type StreamState,
  type SupportConversation,
  type SupportMessage
} from "@/services/support_api";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";

import styles from "./support_page.module.css";

const THREAD_API: SupportThreadApi = {
  fetchThread: fetchMyThread,
  send: sendMyMessage,
  markRead: markMyThreadRead
};

type PageState = "loading" | "error" | "ineligible" | "ready";

// SupportPage is the support chat for clients and trainers: their
// conversations with RYZE support, a form to start one, and the open thread
// with live replies. On phones the list and the thread are separate views.
export const SupportPage = () => {
  const { t, formatDate, intlLocale } = useI18n();
  const history = useHistory();
  const location = useLocation();
  const subjectId = useId();
  const messageId = useId();
  const [pageState, setPageState] = useState<PageState>("loading");
  const [isTrainer, setIsTrainer] = useState(false);
  const [conversations, setConversations] = useState<SupportConversation[]>([]);
  const [streamState, setStreamState] = useState<StreamState>("connecting");
  const [composing, setComposing] = useState(false);
  const [subject, setSubject] = useState("");
  const [firstMessage, setFirstMessage] = useState("");
  const [starting, setStarting] = useState(false);
  const [startError, setStartError] = useState("");
  const startKey = useRef(newClientMessageId());

  const selectedId = new URLSearchParams(location.search).get("conversation");
  const chat = useSupportThread(selectedId, THREAD_API, "user");
  const receiveRef = useRef(chat.receive);
  receiveRef.current = chat.receive;
  const reloadThreadRef = useRef(chat.load);
  reloadThreadRef.current = chat.load;

  const loadedOnce = useRef(false);
  const loadList = useCallback(async () => {
    try {
      setConversations(await fetchMyConversations());
      loadedOnce.current = true;
      setPageState("ready");
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      if (error instanceof ApiError && error.code === "SUPPORT_NOT_AVAILABLE") {
        setPageState("ineligible");
        return;
      }
      // A failed background refresh (for example while offline) keeps what
      // is on screen; the stream refreshes again when it reconnects.
      if (!loadedOnce.current) {
        setPageState("error");
      }
    }
  }, [history]);

  useEffect(() => {
    let cancelled = false;
    fetchSupportEligibility()
      .then((eligibility) => {
        if (cancelled) {
          return;
        }
        setIsTrainer(eligibility.role === "trainer");
        if (eligibility.eligible) {
          void loadList();
        } else {
          setPageState("ineligible");
        }
      })
      .catch((error) => {
        if (cancelled) {
          return;
        }
        if (error instanceof ApiError && error.status === 401) {
          history.replace("/login");
          return;
        }
        setPageState("error");
      });
    return () => {
      cancelled = true;
    };
  }, [history, loadList]);

  // One live stream for the page. Each (re)connection refreshes the list and
  // the open thread, so nothing stored while disconnected is missed.
  useEffect(() => {
    if (pageState !== "ready") {
      return;
    }
    let refreshTimer = 0;
    const close = openSupportStream(
      "/me/support/stream",
      (message: SupportMessage) => {
        receiveRef.current(message);
        window.clearTimeout(refreshTimer);
        refreshTimer = window.setTimeout(() => void loadList(), 400);
      },
      (state) => {
        setStreamState(state);
        if (state === "live") {
          void loadList();
          void reloadThreadRef.current();
        }
      }
    );
    return () => {
      window.clearTimeout(refreshTimer);
      close();
    };
  }, [pageState, loadList]);

  // Keep the list's unread count in step with what the open thread has read.
  useEffect(() => {
    if (!selectedId) {
      return;
    }
    setConversations((current) =>
      current.map((conversation) => (conversation.id === selectedId ? { ...conversation, unread: 0 } : conversation))
    );
  }, [selectedId, chat.thread?.messages.length]);

  // Opening a thread adds a history entry, so the phone's Back returns to the
  // list; the in-page back link replaces it.
  const select = (id: string | null) => {
    if (id) {
      history.push(`/support?conversation=${encodeURIComponent(id)}`);
    } else {
      history.replace("/support");
    }
  };

  const start = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (starting) {
      return;
    }
    const cleanSubject = subject.trim();
    const cleanMessage = firstMessage.trim();
    if (cleanSubject.length < 3 || cleanSubject.length > 120 || !cleanMessage) {
      setStartError(t("support.invalid"));
      return;
    }
    setStarting(true);
    setStartError("");
    try {
      const thread = await openConversation(cleanSubject, cleanMessage, startKey.current);
      startKey.current = newClientMessageId();
      setSubject("");
      setFirstMessage("");
      setComposing(false);
      await loadList();
      select(thread.conversation.id);
    } catch (error) {
      if (error instanceof ApiError && error.code === "TOO_MANY_OPEN_CONVERSATIONS") {
        setStartError(t("support.tooManyOpen"));
      } else if (error instanceof ApiError && error.status === 400) {
        setStartError(t("support.invalid"));
      } else {
        setStartError(t("common.genericError"));
      }
    } finally {
      setStarting(false);
    }
  };

  const labels: SupportThreadLabels = useMemo(
    () => ({
      you: t("support.thread.you"),
      them: t("support.thread.them"),
      placeholder: t("support.thread.placeholder"),
      send: t("support.thread.send"),
      sending: t("support.thread.sending"),
      failed: t("support.thread.failed"),
      retry: t("support.thread.retry"),
      older: t("support.thread.older"),
      loadingOlder: t("support.thread.loadingOlder"),
      closed: t("support.thread.closed"),
      composerLabel: t("support.thread.composerLabel"),
      remaining: (count: number) => t("support.thread.remaining", { count }),
      empty: t("support.thread.empty")
    }),
    [t]
  );

  const formatTime = useCallback(
    (iso: string) =>
      new Date(iso).toLocaleString(intlLocale, { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" }),
    [intlLocale]
  );

  const selectedConversation = chat.thread?.conversation ?? conversations.find((item) => item.id === selectedId) ?? null;

  return (
    <PageWrapper className={styles.page}>

      <div className={styles.main}>
        <header className={styles.intro}>
          <span className={styles.introIcon} aria-hidden="true">
            <LifeBuoy />
          </span>
          <div>
            <h1 className={styles.title}>{t("support.title")}</h1>
            <p className={styles.lead}>{t("support.lead")}</p>
          </div>
        </header>

        {isTrainer ? (
          <Link className={styles.trainerBack} to="/trainer/dashboard">
            <ArrowLeft aria-hidden="true" />
            {t("support.trainerBack")}
          </Link>
        ) : null}

        {pageState === "loading" ? <p className={styles.state}>{t("support.loading")}</p> : null}
        {pageState === "error" ? (
          <p className={styles.errorText} role="alert">
            {t("support.loadError")}
          </p>
        ) : null}

        {pageState === "ineligible" ? (
          <section className={styles.card}>
            <h2 className={styles.cardTitle}>{t("support.notEligibleTitle")}</h2>
            <p className={styles.cardText}>{t("support.notEligible")}</p>
            <Link className={styles.primaryLink} to="/services/generic-program">
              {t("support.browse")}
            </Link>
          </section>
        ) : null}

        {pageState === "ready" ? (
          <div className={styles.layout} data-view={selectedId ? "thread" : "list"}>
            <section className={styles.listPane} aria-label={t("support.conversations")}>
              <button
                type="button"
                className={styles.newButton}
                aria-expanded={composing}
                onClick={() => setComposing((current) => !current)}
              >
                <MessageSquarePlus aria-hidden="true" />
                {t("support.newConversation")}
              </button>

              {composing ? (
                <form className={styles.newForm} onSubmit={start} noValidate>
                  <label className={styles.field} htmlFor={subjectId}>
                    <span>{t("support.subject")}</span>
                    <input
                      id={subjectId}
                      value={subject}
                      maxLength={120}
                      onChange={(event) => setSubject(event.target.value)}
                      aria-describedby={`${subjectId}-hint`}
                    />
                    <small id={`${subjectId}-hint`}>{t("support.subjectHint")}</small>
                  </label>
                  <label className={styles.field} htmlFor={messageId}>
                    <span>{t("support.message")}</span>
                    <textarea
                      id={messageId}
                      value={firstMessage}
                      rows={4}
                      maxLength={4000}
                      onChange={(event) => setFirstMessage(event.target.value)}
                    />
                  </label>
                  {startError ? (
                    <p className={styles.errorText} role="alert">
                      {startError}
                    </p>
                  ) : null}
                  <button type="submit" className={styles.primaryButton} disabled={starting}>
                    {starting ? t("support.starting") : t("support.start")}
                  </button>
                </form>
              ) : null}

              <h2 className={styles.listTitle}>{t("support.conversations")}</h2>
              {conversations.length === 0 ? <p className={styles.state}>{t("support.noConversations")}</p> : null}
              <ul className={styles.list}>
                {conversations.map((conversation) => (
                  <li key={conversation.id}>
                    <button
                      type="button"
                      className={joinClassNames(styles.item, conversation.id === selectedId && styles.itemActive)}
                      aria-current={conversation.id === selectedId ? "true" : undefined}
                      onClick={() => select(conversation.id)}
                    >
                      <span className={styles.itemSubject}>{conversation.subject}</span>
                      <span className={styles.itemMeta}>
                        <span className={conversation.status === "open" ? styles.badgeOpen : styles.badgeClosed}>
                          {conversation.status === "open" ? t("support.open") : t("support.closed")}
                        </span>
                        <time dateTime={conversation.last_message_at}>
                          {formatDate(conversation.last_message_at, { day: "numeric", month: "short" })}
                        </time>
                        {conversation.unread ? (
                          <span className={styles.unread}>{t("support.unread", { count: conversation.unread })}</span>
                        ) : null}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
              <p className={styles.note}>{t("support.notCoaching")}</p>
            </section>

            <section className={styles.threadPane} aria-label={selectedConversation?.subject ?? t("support.title")}>
              {selectedId ? (
                <>
                  <header className={styles.threadHead}>
                    <button type="button" className={styles.back} onClick={() => select(null)}>
                      <ArrowLeft aria-hidden="true" />
                      {t("support.back")}
                    </button>
                    <h2 className={styles.threadTitle}>{selectedConversation?.subject ?? ""}</h2>
                    <span className={styles.live} data-state={streamState} role="status">
                      {streamState === "live"
                        ? t("support.live")
                        : streamState === "offline"
                          ? t("support.offline")
                          : t("support.reconnecting")}
                    </span>
                  </header>
                  {chat.status === "error" ? (
                    <p className={styles.errorText} role="alert">
                      {t("support.loadError")}
                    </p>
                  ) : chat.thread ? (
                    <SupportThread
                      key={chat.thread.conversation.id}
                      perspective="user"
                      messages={chat.thread.messages}
                      pending={chat.pending}
                      closed={chat.thread.conversation.status === "closed"}
                      hasOlder={chat.thread.has_older}
                      loadingOlder={chat.loadingOlder}
                      onLoadOlder={() => void chat.loadOlder()}
                      onSend={chat.send}
                      onRetry={chat.retry}
                      labels={labels}
                      formatTime={formatTime}
                    />
                  ) : (
                    <p className={styles.state}>{t("common.loading")}</p>
                  )}
                </>
              ) : (
                <p className={styles.choose}>{t("support.choose")}</p>
              )}
            </section>
          </div>
        ) : null}
      </div>
    </PageWrapper>
  );
};

export default SupportPage;
