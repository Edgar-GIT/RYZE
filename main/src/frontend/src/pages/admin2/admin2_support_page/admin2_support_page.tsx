import { ArrowLeft, LifeBuoy, Lock, RefreshCw, Unlock } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { Admin2EmptyState } from "@/components/admin2/admin2_empty_state/admin2_empty_state";
import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Pagination } from "@/components/admin2/admin2_pagination/admin2_pagination";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Button } from "@/components/button/button";
import { SupportThread, type SupportThreadLabels } from "@/components/support_chat/support_thread";
import { useSupportThread, type SupportThreadApi } from "@/components/support_chat/use_support_thread";
import {
  fetchStaffConversations,
  fetchStaffThread,
  markStaffThreadRead,
  openSupportStream,
  sendStaffReply,
  setStaffThreadStatus,
  type StaffConversationList,
  type StreamState,
  type SupportStatus
} from "@/services/support_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./admin2_support_page.module.css";

const PAGE_SIZE = 20;

const TABS: Array<{ status: SupportStatus; label: string }> = [
  { status: "open", label: "Open" },
  { status: "closed", label: "Closed" }
];

const THREAD_API: SupportThreadApi = {
  fetchThread: fetchStaffThread,
  send: sendStaffReply,
  markRead: markStaffThreadRead
};

const formatDateTime = (value: string) =>
  new Date(value).toLocaleString(undefined, { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" });

const STREAM_LABEL: Record<StreamState, string> = {
  connecting: "Connecting…",
  live: "Live",
  reconnecting: "Reconnecting…",
  offline: "Live updates paused — refresh the list"
};

// Admin2SupportPage is the support inbox of the Management Administrator:
// conversations opened by clients and trainers, answered as "RYZE support".
export default function Admin2SupportPage() {
  const [status, setStatus] = useState<SupportStatus>("open");
  const [page, setPage] = useState(1);
  const [list, setList] = useState<StaffConversationList | null>(null);
  const [listError, setListError] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [streamState, setStreamState] = useState<StreamState>("connecting");
  const [statusBusy, setStatusBusy] = useState(false);
  const [actionError, setActionError] = useState("");

  const chat = useSupportThread(selectedId, THREAD_API, "staff");
  const receiveRef = useRef(chat.receive);
  receiveRef.current = chat.receive;
  const reloadThreadRef = useRef(chat.load);
  reloadThreadRef.current = chat.load;

  const loadList = useCallback(async () => {
    try {
      setList(await fetchStaffConversations(status, page, PAGE_SIZE));
      setListError("");
    } catch {
      setListError("Conversations could not be loaded. Try again.");
    }
  }, [status, page]);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  const loadListRef = useRef(loadList);
  loadListRef.current = loadList;

  useEffect(() => {
    let refreshTimer = 0;
    const close = openSupportStream(
      "/admin/support/stream",
      (message) => {
        receiveRef.current(message);
        window.clearTimeout(refreshTimer);
        refreshTimer = window.setTimeout(() => void loadListRef.current(), 400);
      },
      (state) => {
        setStreamState(state);
        if (state === "live") {
          void loadListRef.current();
          void reloadThreadRef.current();
        }
      }
    );
    return () => {
      window.clearTimeout(refreshTimer);
      close();
    };
  }, []);

  const changeStatus = async (next: SupportStatus) => {
    if (!selectedId) {
      return;
    }
    setStatusBusy(true);
    setActionError("");
    try {
      const conversation = await setStaffThreadStatus(selectedId, next);
      chat.setThread((current) => (current ? { ...current, conversation: { ...current.conversation, ...conversation } } : current));
      await loadList();
    } catch {
      setActionError("The conversation could not be updated. Try again.");
    } finally {
      setStatusBusy(false);
    }
  };

  const conversation = chat.thread?.conversation ?? null;
  const requester = chat.thread?.requester;
  const requesterName = requester ? `${requester.first_name} ${requester.last_name}`.trim() || requester.email : "Member";

  const labels: SupportThreadLabels = {
    you: "RYZE support (you)",
    them: requesterName,
    placeholder: "Write a reply as RYZE support",
    send: "Send reply",
    sending: "Sending…",
    failed: "Not sent",
    retry: "Try again",
    older: "Show earlier messages",
    loadingOlder: "Loading…",
    closed: "This conversation is closed. A reply is still delivered; reopen it to let the member write again.",
    composerLabel: "Reply",
    remaining: (count) => `${count} characters left`,
    empty: "No messages yet."
  };

  return (
    <>
      <Admin2PageHeader
        eyebrow="Engagement"
        title="Support"
        description="Questions from clients and trainers. Replies are signed as RYZE support; support is never coaching."
      />

      <div className={styles.layout} data-view={selectedId ? "thread" : "list"}>
        <Admin2Section
          title="Conversations"
          subtitle={list ? `${list.unread} unread message${list.unread === 1 ? "" : "s"} in open conversations` : undefined}
        >
          <div className={styles.toolbar}>
            <div className={styles.tabs} role="tablist" aria-label="Conversation status">
              {TABS.map((tab) => (
                <button
                  key={tab.status}
                  type="button"
                  role="tab"
                  aria-selected={status === tab.status}
                  className={joinClassNames(styles.tab, status === tab.status && styles.tabActive)}
                  onClick={() => {
                    setStatus(tab.status);
                    setPage(1);
                  }}
                >
                  {tab.label}
                </button>
              ))}
            </div>
            <span className={styles.live} data-state={streamState} role="status">
              {STREAM_LABEL[streamState]}
            </span>
          </div>

          {listError ? (
            <div className={styles.pageState}>
              <p className={styles.error}>{listError}</p>
              <Button variant="secondary" size="small" icon={<RefreshCw size={15} />} onClick={() => void loadList()}>
                Try again
              </Button>
            </div>
          ) : list === null ? (
            <div className={styles.pageState}>
              <p className={styles.muted}>Loading conversations…</p>
            </div>
          ) : list.conversations.length === 0 ? (
            <Admin2EmptyState
              icon={LifeBuoy}
              title={status === "open" ? "No open conversations" : "No closed conversations"}
              message={status === "open" ? "New questions from members appear here." : "Closed conversations are listed here."}
            />
          ) : (
            <>
              <ul className={styles.list}>
                {list.conversations.map((item) => (
                  <li key={item.id}>
                    <button
                      type="button"
                      className={joinClassNames(styles.item, item.id === selectedId && styles.itemActive)}
                      aria-current={item.id === selectedId ? "true" : undefined}
                      onClick={() => setSelectedId(item.id)}
                    >
                      <span className={styles.itemTop}>
                        <strong className={styles.subject}>{item.subject}</strong>
                        {item.unread ? <span className={styles.unread}>{item.unread} new</span> : null}
                      </span>
                      <span className={styles.itemMeta}>
                        <span>
                          {item.requester ? `${item.requester.first_name} ${item.requester.last_name}`.trim() : "Member"}
                        </span>
                        <span className={styles.role}>{item.requester_role}</span>
                        <time dateTime={item.last_message_at}>{formatDateTime(item.last_message_at)}</time>
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
              <Admin2Pagination
                page={page}
                totalPages={Math.max(1, Math.ceil(list.total / PAGE_SIZE))}
                pageSize={PAGE_SIZE}
                totalItems={list.total}
                onChange={setPage}
              />
            </>
          )}
        </Admin2Section>

        <section className={styles.threadPane} aria-label={conversation?.subject ?? "Conversation"}>
          {selectedId ? (
            <>
              <header className={styles.threadHead}>
                <button type="button" className={styles.back} onClick={() => setSelectedId(null)}>
                  <ArrowLeft aria-hidden="true" />
                  All conversations
                </button>
                <div className={styles.threadTitleBlock}>
                  <h2 className={styles.threadTitle}>{conversation?.subject ?? "Loading…"}</h2>
                  {requester ? (
                    <p className={styles.muted}>
                      {requesterName} · {requester.email} · {conversation?.requester_role}
                    </p>
                  ) : null}
                </div>
                {conversation ? (
                  conversation.status === "open" ? (
                    <Button
                      variant="secondary"
                      size="small"
                      icon={<Lock size={15} />}
                      disabled={statusBusy}
                      onClick={() => void changeStatus("closed")}
                    >
                      Close
                    </Button>
                  ) : (
                    <Button
                      variant="secondary"
                      size="small"
                      icon={<Unlock size={15} />}
                      disabled={statusBusy}
                      onClick={() => void changeStatus("open")}
                    >
                      Reopen
                    </Button>
                  )
                ) : null}
              </header>
              {actionError ? (
                <p className={styles.error} role="alert">
                  {actionError}
                </p>
              ) : null}
              {chat.status === "error" ? (
                <p className={styles.error} role="alert">
                  The conversation could not be loaded.
                </p>
              ) : chat.thread ? (
                <SupportThread
                  key={chat.thread.conversation.id}
                  perspective="staff"
                  messages={chat.thread.messages}
                  pending={chat.pending}
                  closed={chat.thread.conversation.status === "closed"}
                  canWriteWhenClosed
                  hasOlder={chat.thread.has_older}
                  loadingOlder={chat.loadingOlder}
                  onLoadOlder={() => void chat.loadOlder()}
                  onSend={chat.send}
                  onRetry={chat.retry}
                  labels={labels}
                  formatTime={formatDateTime}
                />
              ) : (
                <p className={styles.muted}>Loading conversation…</p>
              )}
            </>
          ) : (
            <p className={styles.muted}>Choose a conversation to read and reply.</p>
          )}
        </section>
      </div>
    </>
  );
}
