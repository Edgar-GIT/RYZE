import { RotateCw, Send } from "lucide-react";
import type { FormEvent, KeyboardEvent } from "react";
import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";

import type { SupportMessage } from "@/services/support_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./support_thread.module.css";

export const MAX_MESSAGE_LENGTH = 4000;

// A message the reader sent that the server has not confirmed yet.
export interface PendingMessage {
  clientId: string;
  body: string;
  state: "sending" | "failed";
}

export interface SupportThreadLabels {
  you: string;
  them: string;
  placeholder: string;
  send: string;
  sending: string;
  failed: string;
  retry: string;
  older: string;
  loadingOlder: string;
  closed: string;
  composerLabel: string;
  remaining: (count: number) => string;
  empty: string;
}

interface SupportThreadProps {
  // Which side the reader is on: "user" (client or trainer) or "staff".
  perspective: "user" | "staff";
  messages: SupportMessage[];
  pending: PendingMessage[];
  closed: boolean;
  // Whether the reader may still write while the conversation is closed.
  canWriteWhenClosed?: boolean;
  hasOlder: boolean;
  loadingOlder: boolean;
  onLoadOlder: () => void;
  onSend: (body: string) => void;
  onRetry: (clientId: string) => void;
  labels: SupportThreadLabels;
  formatTime: (iso: string) => string;
}

// SupportThread renders one support conversation: messages oldest first, the
// reader's own messages on the right, and the composer. It keeps the view
// pinned to the newest message unless the reader scrolled up to read history.
export const SupportThread = ({
  perspective,
  messages,
  pending,
  closed,
  canWriteWhenClosed = false,
  hasOlder,
  loadingOlder,
  onLoadOlder,
  onSend,
  onRetry,
  labels,
  formatTime
}: SupportThreadProps) => {
  const composerId = useId();
  const listRef = useRef<HTMLDivElement>(null);
  const pinnedRef = useRef(true);
  const [draft, setDraft] = useState("");

  const onScroll = () => {
    const list = listRef.current;
    if (list) {
      pinnedRef.current = list.scrollHeight - list.scrollTop - list.clientHeight < 80;
    }
  };

  // Keep the newest message in view when it arrives (unless reading history).
  const newestKey = `${messages[messages.length - 1]?.id ?? ""}:${pending.length}`;
  useLayoutEffect(() => {
    const list = listRef.current;
    if (list && pinnedRef.current) {
      list.scrollTop = list.scrollHeight;
    }
  }, [newestKey]);

  useEffect(() => {
    pinnedRef.current = true;
  }, []);

  const writable = !closed || canWriteWhenClosed;
  const trimmed = draft.trim();

  const submit = (event?: FormEvent) => {
    event?.preventDefault();
    if (!trimmed || trimmed.length > MAX_MESSAGE_LENGTH) {
      return;
    }
    pinnedRef.current = true;
    onSend(trimmed);
    setDraft("");
  };

  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    // Enter adds a line on phones; Ctrl/Cmd + Enter sends everywhere.
    if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) {
      submit();
    }
  };

  const mine = (message: SupportMessage) => message.author === perspective;

  // Announce a message from the other side that arrived after the thread
  // was first shown (the first render only records where the history ends).
  const lastAnnounced = useRef<number | null>(null);
  const [announcement, setAnnouncement] = useState("");
  const newestOther = [...messages].reverse().find((message) => message.author !== perspective);
  useEffect(() => {
    if (!newestOther) {
      if (lastAnnounced.current === null) {
        lastAnnounced.current = 0;
      }
      return;
    }
    if (lastAnnounced.current === null) {
      lastAnnounced.current = newestOther.seq;
      return;
    }
    if (newestOther.seq > lastAnnounced.current) {
      lastAnnounced.current = newestOther.seq;
      setAnnouncement(`${labels.them}: ${newestOther.body}`);
    }
  }, [newestOther, labels.them]);
  const remaining = MAX_MESSAGE_LENGTH - draft.length;

  return (
    <div className={styles.thread}>
      {/* Only new messages from the other side are announced, never the
          history or the reader's own messages. */}
      <p className={styles.srOnly} aria-live="polite">
        {announcement}
      </p>
      <div className={styles.list} ref={listRef} onScroll={onScroll}>
        {hasOlder ? (
          <button type="button" className={styles.older} onClick={onLoadOlder} disabled={loadingOlder}>
            {loadingOlder ? labels.loadingOlder : labels.older}
          </button>
        ) : null}
        {messages.length === 0 && pending.length === 0 ? <p className={styles.empty}>{labels.empty}</p> : null}
        {messages.map((message) => (
          <article key={message.id} className={joinClassNames(styles.message, mine(message) ? styles.mine : styles.theirs)}>
            <header className={styles.meta}>
              <span className={styles.author}>{mine(message) ? labels.you : labels.them}</span>
              <time dateTime={message.created_at}>{formatTime(message.created_at)}</time>
            </header>
            <p className={styles.body}>{message.body}</p>
          </article>
        ))}
        {pending.map((message) => (
          <article key={message.clientId} className={joinClassNames(styles.message, styles.mine, styles.pending)}>
            <header className={styles.meta}>
              <span className={styles.author}>{labels.you}</span>
              <span>{message.state === "sending" ? labels.sending : labels.failed}</span>
            </header>
            <p className={styles.body}>{message.body}</p>
            {message.state === "failed" ? (
              <button type="button" className={styles.retry} onClick={() => onRetry(message.clientId)}>
                <RotateCw aria-hidden="true" />
                {labels.retry}
              </button>
            ) : null}
          </article>
        ))}
      </div>

      {writable ? (
        <form className={styles.composer} onSubmit={submit}>
          {closed ? <p className={styles.closedNote}>{labels.closed}</p> : null}
          <label className={styles.srOnly} htmlFor={composerId}>
            {labels.composerLabel}
          </label>
          <textarea
            id={composerId}
            className={styles.input}
            value={draft}
            maxLength={MAX_MESSAGE_LENGTH}
            rows={2}
            placeholder={labels.placeholder}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={onKeyDown}
          />
          <div className={styles.composerRow}>
            <span className={styles.counter}>{remaining < 500 ? labels.remaining(remaining) : ""}</span>
            <button type="submit" className={styles.send} disabled={!trimmed}>
              <Send aria-hidden="true" />
              {labels.send}
            </button>
          </div>
        </form>
      ) : (
        <p className={styles.closedBanner}>{labels.closed}</p>
      )}
    </div>
  );
};
