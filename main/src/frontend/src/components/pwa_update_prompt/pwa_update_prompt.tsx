import { RefreshCw } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { useI18n } from "@/i18n/i18n";

import styles from "./pwa_update_prompt.module.css";

// How often an open tab checks for a new deployment of the worker.
const UPDATE_CHECK_MS = 60 * 60 * 1000;

// PwaUpdatePrompt registers the service worker (production builds only) and,
// when a new version is waiting, offers to reload into it. The reload is
// never forced: a form being filled in is not lost.
export const PwaUpdatePrompt = () => {
  const { t } = useI18n();
  const [waiting, setWaiting] = useState<ServiceWorker | null>(null);
  const reloading = useRef(false);

  useEffect(() => {
    if (!import.meta.env.PROD || !("serviceWorker" in navigator)) {
      return;
    }
    let timer = 0;
    const onControllerChange = () => {
      if (reloading.current) {
        window.location.reload();
      }
    };
    navigator.serviceWorker.addEventListener("controllerchange", onControllerChange);

    navigator.serviceWorker
      .register("/sw.js", { scope: "/" })
      .then((registration) => {
        const track = (worker: ServiceWorker | null) => {
          if (!worker) {
            return;
          }
          worker.addEventListener("statechange", () => {
            // A worker that installed while another controls the page is an
            // update; on the very first visit there is nothing to replace.
            if (worker.state === "installed" && navigator.serviceWorker.controller) {
              setWaiting(worker);
            }
          });
        };
        if (registration.waiting && navigator.serviceWorker.controller) {
          setWaiting(registration.waiting);
        }
        track(registration.installing);
        registration.addEventListener("updatefound", () => track(registration.installing));
        timer = window.setInterval(() => {
          registration.update().catch(() => undefined);
        }, UPDATE_CHECK_MS);
      })
      .catch(() => {
        // The app works the same without the worker.
      });

    return () => {
      window.clearInterval(timer);
      navigator.serviceWorker.removeEventListener("controllerchange", onControllerChange);
    };
  }, []);

  if (!waiting) {
    return null;
  }

  return (
    <div className={styles.prompt} role="status">
      <p className={styles.text}>{t("pwa.updateAvailable")}</p>
      <button
        type="button"
        className={styles.action}
        onClick={() => {
          reloading.current = true;
          waiting.postMessage("SKIP_WAITING");
        }}
      >
        <RefreshCw aria-hidden="true" />
        {t("pwa.reload")}
      </button>
      <button type="button" className={styles.dismiss} onClick={() => setWaiting(null)}>
        {t("pwa.later")}
      </button>
    </div>
  );
};
