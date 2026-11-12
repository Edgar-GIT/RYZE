import { apiGet, apiPost, apiUrl, encodePathSegment } from "@utils/http_client";

export type SupportAuthor = "user" | "staff";
export type SupportStatus = "open" | "closed";

export interface SupportMessage {
  id: string;
  seq: number;
  conversation_id: string;
  author: SupportAuthor;
  body: string;
  client_message_id?: string;
  created_at: string;
}

export interface SupportRequester {
  first_name: string;
  last_name: string;
  email: string;
}

export interface SupportConversation {
  id: string;
  requester_role: "client" | "trainer";
  subject: string;
  status: SupportStatus;
  last_message_at: string;
  closed_at: string | null;
  created_at: string;
  updated_at: string;
  unread?: number;
  last_message_seq?: number;
  requester?: SupportRequester;
}

export interface SupportThread {
  conversation: SupportConversation;
  messages: SupportMessage[];
  has_older: boolean;
  requester?: SupportRequester;
}

export interface SupportEligibility {
  eligible: boolean;
  role: "client" | "trainer" | "";
}

// A client-chosen id makes a resend after a dropped connection idempotent.
export const newClientMessageId = (): string =>
  typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `m-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`;

const conversationPath = (id: string) => `/conversations/${encodePathSegment(id)}`;

// Account (client or trainer) endpoints.
export const fetchSupportEligibility = () => apiGet<SupportEligibility>("/me/support/eligibility");

export const fetchMyConversations = () =>
  apiGet<{ conversations: SupportConversation[] }>("/me/support/conversations").then((data) => data.conversations);

export const openConversation = (subject: string, body: string, clientMessageId: string) =>
  apiPost<SupportThread>("/me/support/conversations", { subject, body, client_message_id: clientMessageId });

export const fetchMyThread = (id: string, before?: number) =>
  apiGet<SupportThread>(`/me/support${conversationPath(id)}${before ? `?before=${before}` : ""}`);

export const sendMyMessage = (id: string, body: string, clientMessageId: string) =>
  apiPost<SupportMessage>(`/me/support${conversationPath(id)}/messages`, { body, client_message_id: clientMessageId });

export const markMyThreadRead = (id: string, seq: number) =>
  apiPost<Record<string, never>>(`/me/support${conversationPath(id)}/read`, { seq });

// Support staff (ADMIN_2) endpoints.
export interface StaffConversationList {
  conversations: SupportConversation[];
  total: number;
  unread: number;
}

export const fetchStaffConversations = (status: SupportStatus, page: number, limit = 20) =>
  apiGet<StaffConversationList>(`/admin/support/conversations?status=${status}&page=${page}&limit=${limit}`);

export const fetchStaffThread = (id: string, before?: number) =>
  apiGet<SupportThread>(`/admin/support${conversationPath(id)}${before ? `?before=${before}` : ""}`);

export const sendStaffReply = (id: string, body: string, clientMessageId: string) =>
  apiPost<SupportMessage>(`/admin/support${conversationPath(id)}/messages`, { body, client_message_id: clientMessageId });

export const markStaffThreadRead = (id: string, seq: number) =>
  apiPost<Record<string, never>>(`/admin/support${conversationPath(id)}/read`, { seq });

export const setStaffThreadStatus = (id: string, status: SupportStatus) =>
  apiPost<SupportConversation>(`/admin/support${conversationPath(id)}/${status === "closed" ? "close" : "reopen"}`, {});

export type StreamState = "connecting" | "live" | "reconnecting" | "offline";

const RECONNECT_BASE_MS = 2_000;
const RECONNECT_MAX_MS = 30_000;
// After this many failed reconnections the page checks whether the session
// (or support access) ended, instead of retrying forever.
const PROBE_AFTER_ATTEMPTS = 3;

// Whether the stream may still be opened: false when the session ended or the
// account no longer has support. A network failure counts as "still allowed".
const streamStillAllowed = async (path: string): Promise<boolean> => {
  const probe = path === "/me/support/stream" ? "/me/support/eligibility" : "/admin/auth/me";
  try {
    const response = await fetch(apiUrl(probe), { credentials: "include" });
    if (response.status === 401 || response.status === 403) {
      return false;
    }
    if (probe === "/me/support/eligibility" && response.ok) {
      const body = (await response.json()) as { data?: { eligible?: boolean } };
      return body.data?.eligible !== false;
    }
    return true;
  } catch {
    return true;
  }
};

/**
 * Opens a live message stream (Server-Sent Events) and keeps it open. The
 * browser reconnects on its own after a dropped connection and resumes from
 * the last message it received (Last-Event-ID). When the browser gives up
 * instead (some network failures close the stream for good), a new stream is
 * opened with backoff, and at once when the device comes back online, resuming
 * from the last message seen. Returns a function that closes the stream.
 */
export const openSupportStream = (
  path: "/me/support/stream" | "/admin/support/stream",
  onMessage: (message: SupportMessage) => void,
  onState: (state: StreamState) => void
): (() => void) => {
  if (typeof EventSource === "undefined") {
    onState("offline");
    return () => undefined;
  }
  let source: EventSource | null = null;
  let lastSeq = 0;
  let attempts = 0;
  let timer = 0;
  let stopped = false;

  const connect = () => {
    if (stopped) {
      return;
    }
    window.clearTimeout(timer);
    source?.close();
    onState(attempts === 0 ? "connecting" : "reconnecting");
    // The first stream starts from now (the page refreshes what it shows when
    // the stream opens); a replacement resumes after the last message seen.
    const url = lastSeq > 0 ? `${path}?after=${lastSeq}` : path;
    const current = new EventSource(apiUrl(url), { withCredentials: true });
    source = current;
    current.addEventListener("message", (event) => {
      const message = event as MessageEvent<string>;
      const seq = Number(message.lastEventId);
      if (Number.isFinite(seq) && seq > lastSeq) {
        lastSeq = seq;
      }
      try {
        onMessage(JSON.parse(message.data) as SupportMessage);
      } catch {
        // A malformed event is ignored; the next one carries its own data.
      }
    });
    current.onopen = () => {
      attempts = 0;
      onState("live");
    };
    current.onerror = () => {
      if (current.readyState !== EventSource.CLOSED) {
        onState("reconnecting");
        return;
      }
      attempts += 1;
      onState(navigator.onLine === false ? "offline" : "reconnecting");
      const delay = Math.min(RECONNECT_MAX_MS, RECONNECT_BASE_MS * 2 ** Math.min(attempts - 1, 4));
      timer = window.setTimeout(async () => {
        if (attempts >= PROBE_AFTER_ATTEMPTS && navigator.onLine !== false && !(await streamStillAllowed(path))) {
          // The session ended or support is no longer available: stop.
          stopped = true;
          onState("offline");
          return;
        }
        connect();
      }, delay);
    };
  };

  const onOnline = () => {
    if (!source || source.readyState === EventSource.CLOSED) {
      attempts = Math.max(attempts, 1);
      connect();
    }
  };
  window.addEventListener("online", onOnline);
  connect();

  return () => {
    stopped = true;
    window.clearTimeout(timer);
    window.removeEventListener("online", onOnline);
    source?.close();
  };
};
