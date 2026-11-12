import { useCallback, useEffect, useRef, useState } from "react";

import { newClientMessageId, type SupportMessage, type SupportThread } from "@/services/support_api";

import type { PendingMessage } from "./support_thread";

export interface SupportThreadApi {
  fetchThread: (id: string, before?: number) => Promise<SupportThread>;
  send: (id: string, body: string, clientMessageId: string) => Promise<SupportMessage>;
  markRead: (id: string, seq: number) => Promise<unknown>;
}

type Status = "idle" | "loading" | "ready" | "error";

const mergeMessages = (current: SupportMessage[], incoming: SupportMessage[]): SupportMessage[] => {
  const byId = new Map(current.map((message) => [message.id, message]));
  for (const message of incoming) {
    byId.set(message.id, message);
  }
  return [...byId.values()].sort((a, b) => a.seq - b.seq);
};

// useSupportThread holds one open conversation: its messages (loaded, paged
// back and pushed by the live stream), the reader's unconfirmed messages and
// the read position. perspective is the reader's side ("user" or "staff").
export const useSupportThread = (conversationId: string | null, api: SupportThreadApi, perspective: "user" | "staff") => {
  const [status, setStatus] = useState<Status>("idle");
  const [thread, setThread] = useState<SupportThread | null>(null);
  const [pending, setPending] = useState<PendingMessage[]>([]);
  const [loadingOlder, setLoadingOlder] = useState(false);
  const apiRef = useRef(api);
  apiRef.current = api;
  const markedRef = useRef(0);
  // The conversation on screen now. A response for any other conversation
  // (a slow request that finished after the reader switched) is dropped, so
  // the thread shown is always the one a reply is sent to.
  const currentIdRef = useRef(conversationId);
  currentIdRef.current = conversationId;
  const isCurrent = (id: string | undefined | null) => Boolean(id) && id === currentIdRef.current;

  const load = useCallback(async () => {
    if (!conversationId) {
      setThread(null);
      setStatus("idle");
      return;
    }
    try {
      const loaded = await apiRef.current.fetchThread(conversationId);
      if (!isCurrent(conversationId) || loaded.conversation.id !== conversationId) {
        return;
      }
      setThread((current) =>
        current && current.conversation.id === loaded.conversation.id
          ? { ...loaded, messages: mergeMessages(current.messages, loaded.messages), has_older: current.has_older && loaded.has_older }
          : loaded
      );
      setStatus("ready");
    } catch {
      if (!isCurrent(conversationId)) {
        return;
      }
      // Keep a thread already on screen when a background refresh fails.
      setStatus((current) => (current === "ready" ? current : "error"));
    }
    // isCurrent reads a ref; the conversation id is the only input.
  }, [conversationId]);

  useEffect(() => {
    setThread(null);
    setPending([]);
    setStatus(conversationId ? "loading" : "idle");
    markedRef.current = 0;
    void load();
  }, [conversationId, load]);

  // A message pushed by the live stream.
  const receive = useCallback(
    (message: SupportMessage) => {
      if (message.conversation_id !== conversationId) {
        return;
      }
      setThread((current) => (current ? { ...current, messages: mergeMessages(current.messages, [message]) } : current));
      if (message.client_message_id) {
        setPending((current) => current.filter((entry) => entry.clientId !== message.client_message_id));
      }
    },
    [conversationId]
  );

  const deliver = useCallback(
    async (clientId: string, body: string) => {
      if (!conversationId) {
        return;
      }
      setPending((current) => current.map((entry) => (entry.clientId === clientId ? { ...entry, state: "sending" } : entry)));
      try {
        const stored = await apiRef.current.send(conversationId, body, clientId);
        setThread((current) =>
          current && current.conversation.id === stored.conversation_id
            ? { ...current, messages: mergeMessages(current.messages, [stored]) }
            : current
        );
        setPending((current) => current.filter((entry) => entry.clientId !== clientId));
      } catch {
        setPending((current) => current.map((entry) => (entry.clientId === clientId ? { ...entry, state: "failed" } : entry)));
      }
    },
    [conversationId]
  );

  const send = useCallback(
    (body: string) => {
      const clientId = newClientMessageId();
      setPending((current) => [...current, { clientId, body, state: "sending" }]);
      void deliver(clientId, body);
    },
    [deliver]
  );

  // A retry reuses the same client id, so a message that did reach the
  // server before the connection dropped is never stored twice.
  const retry = useCallback(
    (clientId: string) => {
      const entry = pending.find((item) => item.clientId === clientId);
      if (entry) {
        void deliver(clientId, entry.body);
      }
    },
    [deliver, pending]
  );

  const loadOlder = useCallback(async () => {
    if (!conversationId || !thread || thread.messages.length === 0) {
      return;
    }
    setLoadingOlder(true);
    try {
      const older = await apiRef.current.fetchThread(conversationId, thread.messages[0].seq);
      setThread((current) =>
        current && current.conversation.id === older.conversation.id && isCurrent(older.conversation.id)
          ? { ...current, messages: mergeMessages(current.messages, older.messages), has_older: older.has_older }
          : current
      );
    } catch {
      // The button stays available for another try.
    } finally {
      setLoadingOlder(false);
    }
  }, [conversationId, thread]);

  // Record the read position when the other side's messages are on screen.
  const newestOther = thread?.messages.filter((message) => message.author !== perspective).at(-1)?.seq ?? 0;
  useEffect(() => {
    if (!conversationId || newestOther <= markedRef.current || document.visibilityState !== "visible") {
      return;
    }
    markedRef.current = newestOther;
    apiRef.current.markRead(conversationId, newestOther).catch(() => {
      markedRef.current = 0;
    });
  }, [conversationId, newestOther]);

  return { status, thread, setThread, pending, loadingOlder, load, receive, send, retry, loadOlder };
};
