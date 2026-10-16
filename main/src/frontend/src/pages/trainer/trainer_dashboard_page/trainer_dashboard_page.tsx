import { useCallback, useEffect, useState } from "react";
import { ChevronRight, Dumbbell, RefreshCw, Users } from "lucide-react";
import { Link } from "react-router-dom";

import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { fetchTrainerClients, fetchTrainerPrograms, fetchTrainerProfile, type TrainerProfile } from "@/services/trainer_api";

import styles from "@/components/trainer/trainer_page.module.css";

interface DashboardData {
  clientCount: number;
  programCount: number;
  profile: TrainerProfile;
}

type State =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; data: DashboardData };

export const TrainerDashboardPage = () => {
  const [state, setState] = useState<State>({ status: "loading" });
  const [retryCount, setRetryCount] = useState(0);

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      // The dashboard totals come from the real paginated endpoints; only the
      // pagination totals are used, never fabricated numbers.
      const [clients, programs, profile] = await Promise.all([
        fetchTrainerClients(1, 1),
        fetchTrainerPrograms(1, 1),
        fetchTrainerProfile()
      ]);
      setState({
        status: "ready",
        data: {
          clientCount: clients.pagination.total,
          programCount: programs.pagination.total,
          profile
        }
      });
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
        <h1 className={styles.stateTitle}>Trainer dashboard</h1>
        <p className={styles.stateText}>Unable to load your dashboard right now.</p>
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

  const { clientCount, programCount, profile } = state.data;
  const displayName = [profile.user.first_name, profile.user.last_name].filter(Boolean).join(" ");

  return (
    <>
      <header className={styles.pageHeading}>
        <div>
          <p className={styles.eyebrow}>Trainer</p>
          <h1 className={styles.title}>{displayName ? `Welcome back, ${displayName}` : "Welcome back"}</h1>
          <p className={styles.subtitle}>
            This is your trainer workspace. Manage the clients you train and the programs you
            publish.
          </p>
        </div>
      </header>

      <div className={styles.grid}>
        <section className={styles.card}>
          <p className={styles.statLabel}>Clients</p>
          <p className={styles.statValue}>{clientCount}</p>
          <p className={styles.cardText}>
            {clientCount === 1 ? "1 client is linked to you." : `${clientCount} clients are linked to you.`}
          </p>
          <span className={styles.actions}>
            <Button to="/trainer/clients" variant="secondary" size="small" icon={<Users size={15} />} iconPosition="left">
              Manage clients
            </Button>
          </span>
        </section>

        <section className={styles.card}>
          <p className={styles.statLabel}>Programs</p>
          <p className={styles.statValue}>{programCount}</p>
          <p className={styles.cardText}>
            {programCount === 1 ? "1 program in your catalogue." : `${programCount} programs in your catalogue.`}
          </p>
          <span className={styles.actions}>
            <Button to="/trainer/programs" variant="secondary" size="small" icon={<Dumbbell size={15} />} iconPosition="left">
              Manage programs
            </Button>
          </span>
        </section>

        <section className={styles.card}>
          <p className={styles.statLabel}>Your profile</p>
          <p className={styles.cardText}>
            {displayName || profile.user.email}
            <br />
            {profile.user.email}
          </p>
          <Link className={styles.rowMeta} to="/trainer/profile">
            View trainer profile <ChevronRight size={13} aria-hidden="true" />
          </Link>
        </section>
      </div>
    </>
  );
};
