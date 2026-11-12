import { Bell } from "lucide-react";
import { useCallback, useEffect, useId, useRef, useState } from "react";
import { useHistory } from "react-router-dom";

import { useI18n, type TranslationKey } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";
import { ApiError, apiGet, apiPost, encodePathSegment } from "@utils/http_client";

import styles from "./notification_bell.module.css";

interface NotificationItem {
  id: string;
  kind: string;
  params: { program?: string } | null;
  link: string | null;
  read_at: string | null;
  created_at: string;
}

interface NotificationList {
  notifications: NotificationItem[];
  unread: number;
}

const KNOWN_KINDS = [
  "access_granted",
  "pl2_intake_received",
  "pl2_trainer_assigned",
  "pl2_request_assigned",
  "pl2_delivered",
  "support_reply",
  "support_new_message"
] as const;

type KnownKind = (typeof KNOWN_KINDS)[number];

const isKnownKind = (kind: string): kind is KnownKind => KNOWN_KINDS.includes(kind as KnownKind);

// The list refreshes while the page is visible; notifications live for 24
// hours, so a minute is fresh enough without a socket.
const REFRESH_MS = 60_000;

const relativeTime = (iso: string, intlLocale: string): string => {
  const seconds = Math.round((new Date(iso).getTime() - Date.now()) / 1000);
  const format = new Intl.RelativeTimeFormat(intlLocale, { numeric: "auto" });
  if (Math.abs(seconds) < 60) {
    return format.format(0, "minute");
  }
  const minutes = Math.round(seconds / 60);
  if (Math.abs(minutes) < 60) {
    return format.format(minutes, "minute");
  }
  return format.format(Math.round(minutes / 60), "hour");
};

// NotificationBell shows the signed-in account's notifications from the last
// 24 hours. It renders nothing for a signed-out visitor.
export const NotificationBell = () => {
  const { t, intlLocale } = useI18n();
  const history = useHistory();
  const panelId = useId();
  const wrapperRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const [list, setList] = useState<NotificationList | null>(null);
  const [failed, setFailed] = useState(false);
  const [hidden, setHidden] = useState(false);
  const hiddenRef = useRef(false);
  const [open, setOpen] = useState(false);

  const load = useCallback(async () => {
    try {
      const response = await apiGet<NotificationList>("/me/notifications");
      setList(response);
      setFailed(false);
    } catch (error) {
      // A signed-out visitor has no notifications to show.
      if (error instanceof ApiError && error.status === 401) {
        hiddenRef.current = true;
        setHidden(true);
        return;
      }
      setFailed(true);
    }
  }, []);

  useEffect(() => {
    void load();
    // A signed-out visitor (401) stops polling for good.
    const timer = window.setInterval(() => {
      if (hiddenRef.current) {
        window.clearInterval(timer);
        return;
      }
      if (document.visibilityState === "visible") {
        void load();
      }
    }, REFRESH_MS);
    const onVisible = () => {
      if (!hiddenRef.current && document.visibilityState === "visible") {
        void load();
      }
    };
    document.addEventListener("visibilitychange", onVisible);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisible);
    };
  }, [load]);

  // Close on Escape (focus returns to the bell) and on a click outside.
  useEffect(() => {
    if (!open) {
      return;
    }
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
        buttonRef.current?.focus();
      }
    };
    const onPointer = (event: PointerEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
  }, [open]);

  if (hidden) {
    return null;
  }

  const unread = list?.unread ?? 0;

  const textFor = (item: NotificationItem): string => {
    const program = item.params?.program || t("notifications.fallbackProgram");
    const key: TranslationKey = isKnownKind(item.kind) ? `notifications.kinds.${item.kind}` : "notifications.kinds.other";
    return t(key, { program });
  };

  const openItem = async (item: NotificationItem) => {
    if (!item.read_at) {
      setList((current) =>
        current
          ? {
              unread: Math.max(0, current.unread - 1),
              notifications: current.notifications.map((entry) =>
                entry.id === item.id ? { ...entry, read_at: new Date().toISOString() } : entry
              )
            }
          : current
      );
      apiPost(`/me/notifications/${encodePathSegment(item.id)}/read`, {}).catch(() => undefined);
    }
    setOpen(false);
    if (item.link && item.link.startsWith("/") && !item.link.startsWith("//")) {
      history.push(item.link);
    }
  };

  const markAll = async () => {
    setList((current) =>
      current
        ? {
            unread: 0,
            notifications: current.notifications.map((entry) => ({
              ...entry,
              read_at: entry.read_at ?? new Date().toISOString()
            }))
          }
        : current
    );
    try {
      await apiPost("/me/notifications/read-all", {});
    } catch {
      void load();
    }
  };

  return (
    <div className={styles.wrapper} ref={wrapperRef}>
      <button
        ref={buttonRef}
        type="button"
        className={styles.bell}
        aria-label={unread > 0 ? t("notifications.openWithUnread", { count: unread }) : t("notifications.open")}
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => {
          setOpen((current) => !current);
          if (!open) {
            void load();
          }
        }}
      >
        <Bell aria-hidden="true" strokeWidth={1.8} />
        {unread > 0 ? (
          <span className={styles.badge} aria-hidden="true">
            {unread > 9 ? "9+" : unread}
          </span>
        ) : null}
      </button>

      {open ? (
        <div id={panelId} className={styles.panel} role="region" aria-label={t("notifications.title")}>
          <div className={styles.head}>
            <p className={styles.title}>{t("notifications.title")}</p>
            {unread > 0 ? (
              <button type="button" className={styles.markAll} onClick={() => void markAll()}>
                {t("notifications.markAll")}
              </button>
            ) : null}
          </div>

          {failed && !list ? <p className={styles.state}>{t("notifications.loadError")}</p> : null}
          {list && list.notifications.length === 0 ? <p className={styles.state}>{t("notifications.empty")}</p> : null}

          {list && list.notifications.length > 0 ? (
            <ul className={styles.list}>
              {list.notifications.map((item) => (
                <li key={item.id}>
                  <button
                    type="button"
                    className={joinClassNames(styles.item, !item.read_at && styles.unread)}
                    onClick={() => void openItem(item)}
                  >
                    <span className={styles.dot} aria-hidden="true" />
                    <span className={styles.itemText}>
                      <span className={styles.message}>{textFor(item)}</span>
                      <time className={styles.time} dateTime={item.created_at}>
                        {relativeTime(item.created_at, intlLocale)}
                      </time>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          ) : null}

          <p className={styles.footnote}>{t("notifications.lifetime")}</p>
        </div>
      ) : null}
    </div>
  );
};
