import { useCallback, useEffect, useState } from "react";
import { RefreshCw, UserRound } from "lucide-react";

import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { fetchTrainerProfile, type TrainerProfile } from "@/services/trainer_api";

import styles from "@/components/trainer/trainer_page.module.css";
import pageStyles from "./trainer_profile_page.module.css";

type State =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; profile: TrainerProfile };

// The trainer profile is a view-only identity card in the current product
// stage: the backend exposes the trainer identity but defines no editable
// trainer fields, so the page must not invent an editing workflow.
export const TrainerProfilePage = () => {
  const [state, setState] = useState<State>({ status: "loading" });
  const [retryCount, setRetryCount] = useState(0);

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const profile = await fetchTrainerProfile();
      setState({ status: "ready", profile });
    } catch {
      setState({ status: "error" });
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load, retryCount]);

  if (state.status === "loading") {
    return <LoadingScreen />;
  }

  if (state.status === "error") {
    return (
      <section className={styles.stateCard}>
        <h1 className={styles.stateTitle}>Trainer profile</h1>
        <p className={styles.stateText}>Unable to load your trainer profile right now.</p>
        <div className={styles.actions}>
          <Button
            variant="secondary"
            size="small"
            icon={<RefreshCw size={15} />}
            onClick={() => setRetryCount((count) => count + 1)}
          >
            Try again
          </Button>
        </div>
      </section>
    );
  }

  const { user } = state.profile;
  const displayName = [user.first_name, user.last_name].filter(Boolean).join(" ") || "Trainer";
  const memberSince = new Date(user.created_at).toLocaleDateString(undefined, {
    year: "numeric",
    month: "long",
    day: "numeric"
  });

  return (
    <>
      <header className={styles.pageHeading}>
        <div>
          <p className={styles.eyebrow}>Trainer</p>
          <h1 className={styles.title}>Trainer profile</h1>
          <p className={styles.subtitle}>
            This is the identity attached to everything you publish on RYZE.
          </p>
        </div>
      </header>

      <section className={styles.card}>
        <div className={pageStyles.identity}>
          <span className={pageStyles.avatar} aria-hidden="true">
            <UserRound strokeWidth={1.6} />
          </span>
          <div className={styles.rowMain}>
            <span className={pageStyles.name}>{displayName}</span>
            <span className={styles.rowMeta}>{user.email}</span>
          </div>
        </div>

        <div className={pageStyles.metaGrid}>
          <div className={styles.rowMain}>
            <span className={styles.statLabel}>Member since</span>
            <span className={styles.rowTitle}>{memberSince}</span>
          </div>
          <div className={styles.rowMain}>
            <span className={styles.statLabel}>Status</span>
            <span className={styles.rowTitle}>Active trainer</span>
          </div>
        </div>

        <p className={styles.hint}>
          Editing the trainer profile (bio, photo, certifications) is not part of RYZE yet. When
          those fields are defined they will appear here.
        </p>
      </section>
    </>
  );
};
