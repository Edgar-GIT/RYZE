import { useCallback, useEffect, useState } from "react";
import { ChevronRight, Plus, RefreshCw } from "lucide-react";
import { Link, useHistory } from "react-router-dom";

import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import {
  createTrainerProgram,
  fetchTrainerPrograms,
  TrainerProgramStatus,
  TrainerProgramType,
  type TrainerProgram,
  type TrainerProgramStatusValue,
  type TrainerProgramTypeValue
} from "@/services/trainer_api";
import { ApiError } from "@utils/http_client";

import styles from "@/components/trainer/trainer_page.module.css";

const PAGE_SIZE = 20;

const MIN_PAID_PRICE_MINOR_UNITS = 100;

type State =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; programs: TrainerProgram[]; page: number; totalPages: number };

const typeLabel = (type: string): string =>
  type === TrainerProgramType.FREE
    ? "Free"
    : type === TrainerProgramType.PREMIUM
      ? "Premium"
      : "Personalized";

const formatPrice = (program: TrainerProgram): string =>
  program.type === TrainerProgramType.FREE
    ? "Free"
    : `${(program.price_minor_units / 100).toFixed(2)} ${program.currency}`;

export const TrainerProgramsPage = () => {
  const history = useHistory();
  const [state, setState] = useState<State>({ status: "loading" });
  const [page, setPage] = useState(1);
  const [retryCount, setRetryCount] = useState(0);

  const [formOpen, setFormOpen] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [type, setType] = useState<TrainerProgramTypeValue>(TrainerProgramType.FREE);
  const [status, setStatus] = useState<TrainerProgramStatusValue>(TrainerProgramStatus.DRAFT);
  const [priceEuros, setPriceEuros] = useState("");
  const [creating, setCreating] = useState(false);
  const [formError, setFormError] = useState("");

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const result = await fetchTrainerPrograms(page, PAGE_SIZE);
      setState({
        status: "ready",
        programs: result.programs,
        page: result.pagination.page,
        totalPages: result.pagination.total_pages
      });
    } catch {
      setState({ status: "error" });
    }
  }, [page]);

  useEffect(() => {
    void load();
  }, [load, retryCount]);

  const isPaid = type !== TrainerProgramType.FREE;

  const create = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      setFormError("Give the program a name.");
      return;
    }

    let priceMinorUnits = 0;
    if (isPaid) {
      const euros = Number.parseFloat(priceEuros.replace(",", "."));
      if (!Number.isFinite(euros) || euros < MIN_PAID_PRICE_MINOR_UNITS / 100) {
        setFormError("Paid programs need a price of at least 1.00 EUR.");
        return;
      }
      priceMinorUnits = Math.round(euros * 100);
    }

    setCreating(true);
    setFormError("");
    try {
      const program = await createTrainerProgram({
        name: trimmedName,
        description: description.trim(),
        type,
        status,
        price_minor_units: priceMinorUnits
      });
      history.push(`/trainer/programs/${program.id}`);
    } catch (error) {
      if (error instanceof ApiError && error.code === "VALIDATION_ERROR") {
        setFormError("The server rejected these program details. Check the name, price and status.");
      } else {
        setFormError("Unable to create the program right now.");
      }
    } finally {
      setCreating(false);
    }
  };

  if (state.status === "loading") {
    return <LoadingScreen />;
  }

  if (state.status === "error") {
    return (
      <section className={styles.stateCard}>
        <h1 className={styles.stateTitle}>Programs</h1>
        <p className={styles.stateText}>Unable to load your programs right now.</p>
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

  const { programs, totalPages } = state;

  return (
    <>
      <header className={styles.pageHeading}>
        <div>
          <p className={styles.eyebrow}>Trainer</p>
          <h1 className={styles.title}>Programs</h1>
          <p className={styles.subtitle}>
            The training programs you publish on RYZE. Only you can edit or delete them.
          </p>
        </div>
        <div className={styles.headingActions}>
          <Button
            variant="primary"
            size="small"
            icon={<Plus size={15} />}
            onClick={() => {
              setFormOpen((open) => !open);
              setFormError("");
            }}
          >
            {formOpen ? "Close" : "New program"}
          </Button>
        </div>
      </header>

      {formOpen ? (
        <section className={styles.card}>
          <h2 className={styles.cardTitle}>Create a program</h2>
          <div className={styles.form}>
            <div className={styles.formRow}>
              <label className={styles.field}>
                <span className={styles.label}>Name</span>
                <input
                  className={styles.input}
                  value={name}
                  maxLength={255}
                  onChange={(event) => setName(event.target.value)}
                />
              </label>
              <label className={styles.field}>
                <span className={styles.label}>Type</span>
                <select
                  className={styles.select}
                  value={type}
                  onChange={(event) => setType(event.target.value as TrainerProgramTypeValue)}
                >
                  <option value={TrainerProgramType.FREE}>Free</option>
                  <option value={TrainerProgramType.PREMIUM}>Premium</option>
                  <option value={TrainerProgramType.PERSONALIZED}>Personalized</option>
                </select>
              </label>
              <label className={styles.field}>
                <span className={styles.label}>Price (EUR)</span>
                <input
                  className={styles.input}
                  inputMode="decimal"
                  value={isPaid ? priceEuros : "0"}
                  disabled={!isPaid}
                  onChange={(event) => setPriceEuros(event.target.value)}
                />
              </label>
              <label className={styles.field}>
                <span className={styles.label}>Initial status</span>
                <select
                  className={styles.select}
                  value={status}
                  onChange={(event) => setStatus(event.target.value as TrainerProgramStatusValue)}
                >
                  <option value={TrainerProgramStatus.DRAFT}>Draft</option>
                  <option value={TrainerProgramStatus.PUBLISHED}>Published</option>
                </select>
              </label>
            </div>
            <label className={styles.field}>
              <span className={styles.label}>Description</span>
              <textarea
                className={styles.textarea}
                value={description}
                maxLength={5000}
                onChange={(event) => setDescription(event.target.value)}
              />
            </label>
            <p className={styles.hint}>
              Free programs are always priced at zero. Paid programs need a price of at least 1.00
              EUR.
            </p>
            {formError ? <p className={styles.error}>{formError}</p> : null}
            <div className={styles.actions}>
              <Button
                variant="primary"
                size="small"
                disabled={creating}
                onClick={() => void create()}
              >
                {creating ? "Creating…" : "Create program"}
              </Button>
            </div>
          </div>
        </section>
      ) : null}

      <section className={styles.sectionGap}>
        {programs.length === 0 ? (
          <div className={styles.stateCard}>
            <h2 className={styles.stateTitle}>No programs yet</h2>
            <p className={styles.stateText}>
              Create your first program, add its weeks and workouts, then publish it when it is
              ready.
            </p>
          </div>
        ) : (
          <div className={styles.list}>
            {programs.map((program) => (
              <Link key={program.id} className={styles.listRow} to={`/trainer/programs/${program.id}`}>
                <div className={styles.rowMain}>
                  <span className={styles.rowTitle}>{program.name}</span>
                  <span className={styles.rowMeta}>
                    {typeLabel(program.type)} · {formatPrice(program)} · updated{" "}
                    {new Date(program.updated_at).toLocaleDateString()}
                  </span>
                </div>
                <div className={styles.rowActions}>
                  <span
                    className={
                      program.status === TrainerProgramStatus.PUBLISHED
                        ? styles.chipPublished
                        : styles.chipDraft
                    }
                  >
                    {program.status === TrainerProgramStatus.PUBLISHED ? "Published" : "Draft"}
                  </span>
                  <ChevronRight size={16} aria-hidden="true" />
                </div>
              </Link>
            ))}
          </div>
        )}

        {totalPages > 1 ? (
          <div className={styles.pagination}>
            <Button
              variant="secondary"
              size="small"
              disabled={page <= 1}
              onClick={() => setPage((current) => Math.max(1, current - 1))}
            >
              Previous
            </Button>
            <span className={styles.pageInfo}>
              Page {page} of {totalPages}
            </span>
            <Button
              variant="secondary"
              size="small"
              disabled={page >= totalPages}
              onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
            >
              Next
            </Button>
          </div>
        ) : null}
      </section>
    </>
  );
};
