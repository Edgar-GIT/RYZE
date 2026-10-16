import { useCallback, useEffect, useState } from "react";
import { RefreshCw, Search, Trash2, UserPlus } from "lucide-react";

import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import {
  addTrainerClient,
  fetchTrainerClients,
  lookupTrainerClient,
  removeTrainerClient,
  type TrainerClient,
  type TrainerClientCandidate
} from "@/services/trainer_api";
import { ApiError } from "@utils/http_client";

import styles from "@/components/trainer/trainer_page.module.css";

const PAGE_SIZE = 20;

interface ClientsPage {
  clients: TrainerClient[];
  page: number;
  totalPages: number;
  total: number;
}

type State =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; data: ClientsPage };

// mapLookupError converts the exact-email lookup failures into client-owned
// wording. Raw server tokens are never displayed.
const mapLookupError = (error: unknown): string => {
  if (error instanceof ApiError) {
    if (error.code === "CLIENT_NOT_FOUND") {
      return "No active RYZE account matches that email.";
    }
    if (error.code === "VALIDATION_ERROR") {
      return "Enter a valid email address.";
    }
  }
  return "Unable to look up that email right now.";
};

export const TrainerClientsPage = () => {
  const [state, setState] = useState<State>({ status: "loading" });
  const [page, setPage] = useState(1);
  const [retryCount, setRetryCount] = useState(0);

  const [email, setEmail] = useState("");
  const [lookupBusy, setLookupBusy] = useState(false);
  const [candidate, setCandidate] = useState<TrainerClientCandidate | null>(null);
  const [addBusy, setAddBusy] = useState(false);
  const [feedback, setFeedback] = useState<{ kind: "error" | "success"; text: string } | null>(null);
  const [confirmRemoveId, setConfirmRemoveId] = useState("");
  const [removeBusy, setRemoveBusy] = useState("");

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const result = await fetchTrainerClients(page, PAGE_SIZE);
      setState({
        status: "ready",
        data: {
          clients: result.clients,
          page: result.pagination.page,
          totalPages: result.pagination.total_pages,
          total: result.pagination.total
        }
      });
    } catch {
      setState({ status: "error" });
    }
  }, [page]);

  useEffect(() => {
    void load();
  }, [load, retryCount]);

  const lookup = async () => {
    const trimmed = email.trim();
    if (!trimmed) {
      setFeedback({ kind: "error", text: "Enter the client's email address first." });
      return;
    }
    setLookupBusy(true);
    setFeedback(null);
    setCandidate(null);
    try {
      const result = await lookupTrainerClient(trimmed);
      setCandidate(result);
    } catch (error) {
      setFeedback({ kind: "error", text: mapLookupError(error) });
    } finally {
      setLookupBusy(false);
    }
  };

  const confirmAdd = async () => {
    if (!candidate) {
      return;
    }
    setAddBusy(true);
    setFeedback(null);
    try {
      const client = await addTrainerClient(candidate.user_id);
      const name = [client.user.first_name, client.user.last_name].filter(Boolean).join(" ");
      setFeedback({ kind: "success", text: `${name || client.user.email} was added to your clients.` });
      setCandidate(null);
      setEmail("");
      await load();
    } catch (error) {
      if (error instanceof ApiError && error.code === "CLIENT_ALREADY_ADDED") {
        setFeedback({ kind: "error", text: "That person is already one of your clients." });
      } else if (error instanceof ApiError && error.code === "VALIDATION_ERROR") {
        setFeedback({ kind: "error", text: "You cannot add yourself as a client." });
      } else {
        setFeedback({ kind: "error", text: "Unable to add that client right now." });
      }
    } finally {
      setAddBusy(false);
    }
  };

  const remove = async (client: TrainerClient) => {
    setRemoveBusy(client.user_id);
    setFeedback(null);
    try {
      await removeTrainerClient(client.user_id);
      setConfirmRemoveId("");
      await load();
    } catch (error) {
      if (error instanceof ApiError && error.code === "CLIENT_RELATION_NOT_FOUND") {
        // The relationship was already removed elsewhere: the list is stale,
        // not broken, so refresh it and say what actually happened.
        setConfirmRemoveId("");
        setFeedback({ kind: "error", text: "That client was already removed." });
        await load();
      } else {
        setFeedback({ kind: "error", text: "Unable to remove that client right now." });
      }
    } finally {
      setRemoveBusy("");
    }
  };

  if (state.status === "loading") {
    return <LoadingScreen />;
  }

  if (state.status === "error") {
    return (
      <section className={styles.stateCard}>
        <h1 className={styles.stateTitle}>Clients</h1>
        <p className={styles.stateText}>Unable to load your clients right now.</p>
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

  const { clients, totalPages } = state.data;

  return (
    <>
      <header className={styles.pageHeading}>
        <div>
          <p className={styles.eyebrow}>Trainer</p>
          <h1 className={styles.title}>Clients</h1>
          <p className={styles.subtitle}>
            The people you train. Adding a client links their existing RYZE account to you; it never
            creates an account for them.
          </p>
        </div>
      </header>

      <section className={styles.card}>
        <h2 className={styles.cardTitle}>Add a client</h2>
        <p className={styles.cardText}>
          Look the client up by the exact email they use on RYZE, confirm who it is, then add them.
        </p>
        <div className={styles.formRow}>
          <label className={styles.field}>
            <span className={styles.label}>Client email</span>
            <input
              type="email"
              className={styles.input}
              value={email}
              placeholder="client@example.com"
              onChange={(event) => {
                setEmail(event.target.value);
                setCandidate(null);
                setFeedback(null);
              }}
              onKeyDown={(event) => {
                if (event.key === "Enter") {
                  event.preventDefault();
                  void lookup();
                }
              }}
            />
          </label>
          <div className={styles.formActionsEnd}>
            <Button
              variant="secondary"
              size="small"
              icon={lookupBusy ? <RefreshCw size={15} className={styles.spinIcon} /> : <Search size={15} />}
              disabled={lookupBusy}
              onClick={() => void lookup()}
            >
              {lookupBusy ? "Searching…" : "Find account"}
            </Button>
          </div>
        </div>

        {candidate ? (
          <div className={styles.listRow}>
            <div className={styles.rowMain}>
              <span className={styles.rowTitle}>
                {[candidate.first_name, candidate.last_name].filter(Boolean).join(" ") || candidate.email}
              </span>
              <span className={styles.rowMeta}>{candidate.email}</span>
            </div>
            <div className={styles.rowActions}>
              <Button
                variant="primary"
                size="small"
                icon={<UserPlus size={15} />}
                disabled={addBusy}
                onClick={() => void confirmAdd()}
              >
                {addBusy ? "Adding…" : "Add as client"}
              </Button>
              <Button variant="ghost" size="small" onClick={() => setCandidate(null)}>
                Cancel
              </Button>
            </div>
          </div>
        ) : null}

        {feedback ? (
          <p className={feedback.kind === "error" ? styles.error : styles.success}>{feedback.text}</p>
        ) : null}
      </section>

      <section className={styles.sectionGap}>
        {clients.length === 0 ? (
          <div className={styles.stateCard}>
            <h2 className={styles.stateTitle}>No clients yet</h2>
            <p className={styles.stateText}>
              When you add a client, they appear here so you can manage their training.
            </p>
          </div>
        ) : (
          <div className={styles.list}>
            {clients.map((client) => {
              const name = [client.user.first_name, client.user.last_name].filter(Boolean).join(" ");
              const confirming = confirmRemoveId === client.user_id;
              return (
                <div key={client.id} className={styles.listRow}>
                  <div className={styles.rowMain}>
                    <span className={styles.rowTitle}>{name || client.user.email}</span>
                    <span className={styles.rowMeta}>
                      {client.user.email} · since {new Date(client.created_at).toLocaleDateString()}
                    </span>
                  </div>
                  <div className={styles.rowActions}>
                    {confirming ? (
                      <>
                        <span className={styles.rowMeta}>Remove this client?</span>
                        <Button
                          variant="danger"
                          size="small"
                          disabled={removeBusy === client.user_id}
                          onClick={() => void remove(client)}
                        >
                          {removeBusy === client.user_id ? "Removing…" : "Confirm"}
                        </Button>
                        <Button variant="ghost" size="small" onClick={() => setConfirmRemoveId("")}>
                          Cancel
                        </Button>
                      </>
                    ) : (
                      <Button
                        variant="ghost"
                        size="small"
                        icon={<Trash2 size={15} />}
                        onClick={() => setConfirmRemoveId(client.user_id)}
                      >
                        Remove
                      </Button>
                    )}
                  </div>
                </div>
              );
            })}
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
