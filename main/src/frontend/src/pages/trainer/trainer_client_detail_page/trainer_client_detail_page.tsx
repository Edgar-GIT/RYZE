import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ArrowLeft,
  ChevronDown,
  ChevronRight,
  ClipboardList,
  Dumbbell,
  RefreshCw,
  Trash2,
  UserPlus
} from "lucide-react";
import { useParams } from "react-router-dom";

import { Button } from "@/components/button/button";
import { AppLoading } from "@/components/app_shell/app_loading";
import {
  assignTrainerClientProgram,
  fetchTrainerClient,
  fetchTrainerClientExerciseHistory,
  fetchTrainerClientOverview,
  fetchTrainerClientPrograms,
  fetchTrainerClientWorkoutSession,
  fetchTrainerClientWorkoutSessions,
  fetchTrainerPrograms,
  removeTrainerClientProgram,
  type TrainerClient,
  type TrainerClientExerciseHistory,
  type TrainerClientOverview,
  type TrainerClientProgramAssignment,
  type TrainerClientSessionSummary,
  type TrainerProgram
} from "@/services/trainer_api";
import type { WorkoutSession } from "@/services/workout_sessions_api";
import { ApiError } from "@utils/http_client";

import styles from "@/components/trainer/trainer_page.module.css";
import pageStyles from "./trainer_client_detail_page.module.css";

const PAGE_SIZE = 10;

type PageState =
  | { status: "loading" }
  | { status: "error" }
  | { status: "not-found" }
  | {
      status: "ready";
      client: TrainerClient;
      overview: TrainerClientOverview;
      assignments: TrainerClientProgramAssignment[];
      programs: TrainerProgram[];
    };

const formatDate = (value: string | null): string => {
  if (!value) {
    return "Not recorded";
  }
  return new Date(value).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric"
  });
};

const formatDateTime = (value: string | null): string => {
  if (!value) {
    return "Not recorded";
  }
  return new Date(value).toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  });
};

const formatDuration = (seconds: number): string => {
  if (seconds <= 0) {
    return "0m";
  }
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (hours === 0) {
    return `${minutes}m`;
  }
  return minutes === 0 ? `${hours}h` : `${hours}h ${minutes}m`;
};

const formatWeight = (value: number | null): string => {
  if (value === null) {
    return "";
  }
  return `${Number(value.toFixed(2))} kg`;
};

const formatPrescribed = (set: {
  prescribed_reps: number | null;
  prescribed_weight_kg: number | null;
}): string => {
  const reps = set.prescribed_reps === null ? "Reps not set" : `${set.prescribed_reps} reps`;
  const weight = formatWeight(set.prescribed_weight_kg);
  return weight ? `${reps} × ${weight}` : reps;
};

const formatActual = (set: {
  actual_reps: number | null;
  actual_weight_kg: number | null;
}): string => {
  const reps = set.actual_reps === null ? "Not logged" : `${set.actual_reps} reps`;
  const weight = formatWeight(set.actual_weight_kg);
  return weight ? `${reps} × ${weight}` : reps;
};

const formatEffort = (set: {
  prescribed_rir: number | null;
  prescribed_rpe: number | null;
  actual_rir: number | null;
  actual_rpe: number | null;
}): string => {
  const parts: string[] = [];
  if (set.actual_rir !== null) {
    parts.push(`RIR ${set.actual_rir}`);
  } else if (set.prescribed_rir !== null) {
    parts.push(`Target RIR ${set.prescribed_rir}`);
  }
  if (set.actual_rpe !== null) {
    parts.push(`RPE ${set.actual_rpe}`);
  } else if (set.prescribed_rpe !== null) {
    parts.push(`Target RPE ${set.prescribed_rpe}`);
  }
  return parts.join(" · ");
};

const sessionStatusLabel = (status: TrainerClientSessionSummary["status"] | WorkoutSession["status"]): string => {
  switch (status) {
    case "in_progress":
      return "In progress";
    case "paused":
      return "Paused";
    case "completed":
      return "Completed";
    case "abandoned":
      return "Ended early";
    default:
      return status;
  }
};

const setStatusLabel = (status: "pending" | "completed" | "skipped"): string => {
  switch (status) {
    case "completed":
      return "Completed";
    case "skipped":
      return "Skipped";
    default:
      return "Not logged";
  }
};

const isNotFound = (error: unknown): boolean =>
  error instanceof ApiError && error.status === 404;

const mapAssignmentError = (error: unknown): string => {
  if (error instanceof ApiError) {
    switch (error.code) {
      case "ASSIGNMENT_ALREADY_ACTIVE":
        return "This client already has a programme assigned by you.";
      case "PROGRAM_NOT_FOUND":
        return "That programme is no longer available.";
      case "CLIENT_RELATION_NOT_FOUND":
        return "This client is no longer linked to you.";
      case "VALIDATION_ERROR":
        return "Choose a programme before assigning it.";
      default:
        break;
    }
  }
  return "The programme could not be updated right now.";
};

export const TrainerClientDetailPage = () => {
  const { userID } = useParams<{ userID: string }>();

  const [state, setState] = useState<PageState>({ status: "loading" });
  const [retryCount, setRetryCount] = useState(0);

  const [selectedProgramId, setSelectedProgramId] = useState("");
  const [assignmentBusy, setAssignmentBusy] = useState("");
  const [confirmRemoveId, setConfirmRemoveId] = useState("");
  const [feedback, setFeedback] = useState<{ kind: "error" | "success"; text: string } | null>(null);

  const [historyLoaded, setHistoryLoaded] = useState(false);
  const [historyPage, setHistoryPage] = useState(1);
  const [history, setHistory] = useState<TrainerClientSessionSummary[]>([]);
  const [historyTotalPages, setHistoryTotalPages] = useState(0);
  const [historyBusy, setHistoryBusy] = useState(false);

  const [expandedSessionId, setExpandedSessionId] = useState("");
  const [sessionDetail, setSessionDetail] = useState<WorkoutSession | null>(null);
  const [sessionDetailBusy, setSessionDetailBusy] = useState("");
  const [sessionDetailError, setSessionDetailError] = useState("");

  const [exerciseId, setExerciseId] = useState("");
  const [exerciseHistory, setExerciseHistory] = useState<TrainerClientExerciseHistory | null>(null);
  const [exerciseBusy, setExerciseBusy] = useState(false);
  const [exerciseError, setExerciseError] = useState("");

  const load = useCallback(async () => {
    setState({ status: "loading" });
    setFeedback(null);
    try {
      const [client, overview, assignments, programs] = await Promise.all([
        fetchTrainerClient(userID),
        fetchTrainerClientOverview(userID),
        fetchTrainerClientPrograms(userID),
        fetchTrainerPrograms(1, 100)
      ]);
      setState({
        status: "ready",
        client,
        overview,
        assignments,
        programs: programs.programs
      });
    } catch (error) {
      setState(isNotFound(error) ? { status: "not-found" } : { status: "error" });
    }
  }, [userID, retryCount]);

  useEffect(() => {
    void load();
  }, [load]);

  const refreshInsights = useCallback(async () => {
    const [overview, assignments] = await Promise.all([
      fetchTrainerClientOverview(userID),
      fetchTrainerClientPrograms(userID)
    ]);
    setState((current) => (current.status === "ready" ? { ...current, overview, assignments } : current));
  }, [userID]);

  const availablePrograms = useMemo(() => {
    if (state.status !== "ready") {
      return [];
    }
    const assignedProgramIds = new Set(
      state.overview.access.assignments
        .filter((assignment) => assignment.owned_by_current_trainer)
        .map((assignment) => assignment.program_id)
    );
    return state.programs.filter((program) => !assignedProgramIds.has(program.id));
  }, [state]);

  const selectedAvailableProgramId = useMemo(
    () =>
      availablePrograms.some((program) => program.id === selectedProgramId)
        ? selectedProgramId
        : availablePrograms[0]?.id ?? "",
    [availablePrograms, selectedProgramId]
  );

  const assignProgram = async () => {
    if (!selectedAvailableProgramId) {
      setFeedback({ kind: "error", text: "Choose a programme first." });
      return;
    }
    setAssignmentBusy("assign");
    setFeedback(null);
    try {
      await assignTrainerClientProgram(userID, selectedAvailableProgramId);
      await refreshInsights();
      setFeedback({ kind: "success", text: "Programme assigned to this client." });
    } catch (error) {
      setFeedback({ kind: "error", text: mapAssignmentError(error) });
    } finally {
      setAssignmentBusy("");
    }
  };

  const removeAssignment = async (assignmentId: string) => {
    setAssignmentBusy(assignmentId);
    setFeedback(null);
    try {
      await removeTrainerClientProgram(userID, assignmentId);
      setConfirmRemoveId("");
      await refreshInsights();
      setFeedback({ kind: "success", text: "Programme assignment removed. The client's training history is unchanged." });
    } catch (error) {
      setFeedback({ kind: "error", text: mapAssignmentError(error) });
    } finally {
      setAssignmentBusy("");
    }
  };

  const loadHistory = async (page: number) => {
    setHistoryBusy(true);
    try {
      const result = await fetchTrainerClientWorkoutSessions(userID, page, PAGE_SIZE);
      setHistory(result.sessions);
      setHistoryPage(result.pagination.page);
      setHistoryTotalPages(result.pagination.total_pages);
      setHistoryLoaded(true);
    } catch (error) {
      setFeedback({ kind: "error", text: isNotFound(error) ? "This client is no longer linked to you." : "Unable to load the training history right now." });
    } finally {
      setHistoryBusy(false);
    }
  };

  const toggleSession = async (sessionId: string) => {
    if (expandedSessionId === sessionId) {
      setExpandedSessionId("");
      setSessionDetail(null);
      setSessionDetailError("");
      return;
    }
    setExpandedSessionId(sessionId);
    setSessionDetail(null);
    setSessionDetailError("");
    setSessionDetailBusy(sessionId);
    try {
      const detail = await fetchTrainerClientWorkoutSession(userID, sessionId);
      setSessionDetail(detail);
    } catch (error) {
      setSessionDetailError(
        isNotFound(error) ? "That session is no longer available." : "Unable to load that session right now."
      );
    } finally {
      setSessionDetailBusy("");
    }
  };

  const loadExerciseHistory = async (selectedExerciseId: string) => {
    setExerciseId(selectedExerciseId);
    setExerciseHistory(null);
    setExerciseError("");
    if (!selectedExerciseId) {
      return;
    }
    setExerciseBusy(true);
    try {
      const result = await fetchTrainerClientExerciseHistory(userID, selectedExerciseId, 20);
      setExerciseHistory(result);
    } catch {
      setExerciseError("Unable to load this exercise history right now.");
    } finally {
      setExerciseBusy(false);
    }
  };

  if (state.status === "loading") {
    return <AppLoading />;
  }

  if (state.status === "not-found") {
    return (
      <section className={styles.stateCard}>
        <h1 className={styles.stateTitle}>Client not found</h1>
        <p className={styles.stateText}>
          This client is not linked to your trainer account, or the account is no longer active.
        </p>
        <div className={styles.actions}>
          <Button to="/trainer/clients" variant="secondary" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
            Back to clients
          </Button>
        </div>
      </section>
    );
  }

  if (state.status === "error") {
    return (
      <section className={styles.stateCard}>
        <h1 className={styles.stateTitle}>Client overview</h1>
        <p className={styles.stateText}>Unable to load this client right now.</p>
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

  const { client, overview } = state;
  const name = [client.user.first_name, client.user.last_name].filter(Boolean).join(" ") || client.user.email;
  const currentAssignment = overview.access.assignments.find((assignment) => assignment.owned_by_current_trainer);
  const otherAssignments = overview.access.assignments.filter((assignment) => !assignment.owned_by_current_trainer);
  const hasTrainingData = overview.statistics.completed_sessions + overview.statistics.abandoned_sessions > 0;
  const displayedSessions = historyLoaded ? history : overview.recent_sessions;

  return (
    <>
      <Button to="/trainer/clients" variant="ghost" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
        All clients
      </Button>

      <header className={`${styles.pageHeading} ${pageStyles.clientHeader}`}>
        <div className={pageStyles.clientIdentity}>
          <span className={pageStyles.avatar} aria-hidden="true">
            {(client.user.first_name[0] ?? client.user.email[0] ?? "?").toUpperCase()}
          </span>
          <div>
            <p className={styles.eyebrow}>Client</p>
            <h1 className={styles.title}>{name}</h1>
            <p className={styles.subtitle}>
              {client.user.email} · client since {formatDate(client.created_at)}
            </p>
          </div>
        </div>
      </header>

      {feedback ? (
        <p className={feedback.kind === "error" ? styles.error : styles.success}>{feedback.text}</p>
      ) : null}

      <section className={styles.sectionGap}>
        <div className={styles.grid}>
          <div className={styles.card}>
            <p className={styles.statLabel}>Current programme</p>
            <p className={styles.statText}>{overview.statistics.current_program_name || "None"}</p>
            <p className={styles.cardText}>
              {overview.statistics.has_active_program
                ? "This is the active programme used for current progress."
                : "Assign a programme or the client can start one through their own access."}
            </p>
          </div>
          <div className={styles.card}>
            <p className={styles.statLabel}>Programme completion</p>
            <p className={styles.statValue}>{Math.round(Number(overview.statistics.completion_percentage) || 0)}%</p>
            <p className={styles.cardText}>
              {overview.statistics.unique_workouts_completed} of {overview.statistics.total_workouts_in_program} workouts completed.
            </p>
          </div>
          <div className={styles.card}>
            <p className={styles.statLabel}>Workouts completed</p>
            <p className={styles.statValue}>{overview.statistics.completed_sessions}</p>
            <p className={styles.cardText}>
              {overview.statistics.abandoned_sessions} ended early · {overview.statistics.total_executions} legacy completions.
            </p>
          </div>
          <div className={styles.card}>
            <p className={styles.statLabel}>Training time logged</p>
            <p className={styles.statValue}>{formatDuration(Number(overview.statistics.total_active_seconds))}</p>
            <p className={styles.cardText}>
              {overview.statistics.completed_sets} sets completed · {overview.statistics.skipped_sets} skipped.
            </p>
          </div>
        </div>
      </section>

      {overview.active_session ? (
        <section className={`${styles.sectionGap} ${styles.card} ${pageStyles.activeCard}`}>
          <div>
            <p className={styles.statLabel}>Live workout</p>
            <h2 className={styles.cardTitle}>{overview.active_session.program_name}</h2>
            <p className={styles.cardText}>
              Week {overview.active_session.week_number} · workout {overview.active_session.workout_position} ·{" "}
              {formatDuration(overview.active_session.elapsed_seconds)} elapsed
            </p>
          </div>
          <span className={`${styles.chip} ${pageStyles.chipActive}`}>
            {sessionStatusLabel(overview.active_session.status)}
          </span>
        </section>
      ) : null}

      <section className={`${styles.sectionGap} ${styles.card}`}>
        <div className={pageStyles.sectionHeading}>
          <div>
            <h2 className={styles.cardTitle}>Programme access</h2>
            <p className={styles.cardText}>Trainer assignments, purchases and Free activations are separate sources of access.</p>
          </div>
        </div>

        {overview.access.assignments.length === 0 && overview.access.purchases.length === 0 && overview.access.free_activations.length === 0 ? (
          <p className={styles.cardText}>This client has no active programme access.</p>
        ) : null}

        {overview.access.assignments.length > 0 ? (
          <div className={styles.list}>
            {overview.access.assignments.map((assignment, index) => (
              <div key={`${assignment.assignment_id ?? assignment.program_id}-assignment-${index}`} className={styles.listRow}>
                <div className={styles.rowMain}>
                  <span className={styles.rowTitle}>{assignment.program_name}</span>
                  <span className={styles.rowMeta}>
                    Trainer assignment · since {formatDate(assignment.assigned_at)}
                    {assignment.owned_by_current_trainer ? " · managed by you" : " · managed by another trainer"}
                  </span>
                </div>
                <div className={styles.rowActions}>
                  <span className={`${styles.chip} ${assignment.access_status === "active" ? pageStyles.chipActive : styles.chipDraft}`}>
                    {assignment.access_status === "active" ? "Active" : "Unavailable"}
                  </span>
                  {assignment.owned_by_current_trainer && assignment.assignment_id ? (
                    confirmRemoveId === assignment.assignment_id ? (
                      <>
                        <span className={styles.rowMeta}>Remove assignment?</span>
                        <Button
                          variant="danger"
                          size="small"
                          disabled={assignmentBusy === assignment.assignment_id}
                          onClick={() => void removeAssignment(assignment.assignment_id as string)}
                        >
                          {assignmentBusy === assignment.assignment_id ? "Removing…" : "Confirm"}
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
                        onClick={() => setConfirmRemoveId(assignment.assignment_id ?? "")}
                      >
                        Remove
                      </Button>
                    )
                  ) : null}
                </div>
              </div>
            ))}
          </div>
        ) : null}

        {otherAssignments.length > 0 ? (
          <p className={styles.hint}>Other trainer assignments are read-only here.</p>
        ) : null}

        {overview.access.purchases.length > 0 ? (
          <div className={styles.list}>
            {overview.access.purchases.map((purchase) => (
              <div key={`${purchase.program_id}-purchase`} className={styles.listRow}>
                <div className={styles.rowMain}>
                  <span className={styles.rowTitle}>{purchase.program_name}</span>
                  <span className={styles.rowMeta}>
                    Purchased access · since {formatDate(purchase.granted_at)}
                    {purchase.expires_at ? ` · expires ${formatDate(purchase.expires_at)}` : " · no expiry"}
                  </span>
                </div>
                <span className={`${styles.chip} ${purchase.access_status === "active" ? pageStyles.chipActive : styles.chipDraft}`}>
                  {purchase.access_status === "active" ? "Active" : "Unavailable"}
                </span>
              </div>
            ))}
          </div>
        ) : null}

        {overview.access.free_activations.length > 0 ? (
          <div className={styles.list}>
            {overview.access.free_activations.map((activation) => (
              <div key={`${activation.program_id}-free`} className={styles.listRow}>
                <div className={styles.rowMain}>
                  <span className={styles.rowTitle}>{activation.program_name}</span>
                  <span className={styles.rowMeta}>
                    Free activation · since {formatDate(activation.granted_at)}
                    {activation.expires_at ? ` · expires ${formatDate(activation.expires_at)}` : ""}
                  </span>
                </div>
                <span className={`${styles.chip} ${activation.access_status === "active" ? pageStyles.chipActive : styles.chipDraft}`}>
                  {activation.access_status === "active" ? "Active" : "Unavailable"}
                </span>
              </div>
            ))}
          </div>
        ) : null}

        <div className={pageStyles.assignmentForm}>
          <label className={styles.field}>
            <span className={styles.label}>Assign one of your programmes</span>
            <select
              className={styles.select}
              value={selectedAvailableProgramId}
              onChange={(event) => setSelectedProgramId(event.target.value)}
              disabled={currentAssignment !== undefined || availablePrograms.length === 0}
            >
              {availablePrograms.length === 0 ? (
                <option value="">No available programmes</option>
              ) : (
                availablePrograms.map((program) => (
                  <option key={program.id} value={program.id}>
                    {program.name} ({program.status})
                  </option>
                ))
              )}
            </select>
          </label>
          <div className={pageStyles.assignmentActions}>
            <Button
              variant="secondary"
              size="small"
              icon={<UserPlus size={15} />}
              iconPosition="left"
              disabled={Boolean(currentAssignment) || availablePrograms.length === 0 || assignmentBusy === "assign"}
              onClick={() => void assignProgram()}
            >
              {assignmentBusy === "assign" ? "Assigning…" : currentAssignment ? "Programme already assigned" : "Assign programme"}
            </Button>
            {availablePrograms.length === 0 ? (
              <Button to="/trainer/programs" variant="ghost" size="small" icon={<ClipboardList size={15} />} iconPosition="left">
                Manage programmes
              </Button>
            ) : null}
          </div>
        </div>
      </section>

      <section className={`${styles.sectionGap} ${styles.card}`}>
        <div className={pageStyles.sectionHeading}>
          <div>
            <h2 className={styles.cardTitle}>Session history</h2>
            <p className={styles.cardText}>
              Full sessions include recorded sets and performance. Legacy completions have no set-level detail.
            </p>
          </div>
          {!historyLoaded && hasTrainingData ? (
            <Button
              variant="secondary"
              size="small"
              disabled={historyBusy}
              onClick={() => void loadHistory(1)}
            >
              {historyBusy ? "Loading…" : "View all sessions"}
            </Button>
          ) : null}
        </div>

        {displayedSessions.length === 0 ? (
          <p className={styles.cardText}>No full workout sessions have been recorded yet.</p>
        ) : (
          <div className={styles.list}>
            {displayedSessions.map((session) => {
              const expanded = expandedSessionId === session.id;
              const detail = expanded ? sessionDetail : null;
              return (
                <div key={session.id} className={pageStyles.sessionItem}>
                  <button
                    type="button"
                    className={pageStyles.sessionButton}
                    onClick={() => void toggleSession(session.id)}
                    aria-expanded={expanded}
                  >
                    <span className={styles.rowMain}>
                      <span className={styles.rowTitle}>{session.program_name}</span>
                      <span className={styles.rowMeta}>
                        Week {session.week_number} · workout {session.workout_position} ·{" "}
                        {formatDateTime(session.completed_at ?? session.abandoned_at ?? session.started_at)}
                      </span>
                    </span>
                    <span className={styles.rowActions}>
                      <span className={`${styles.chip} ${session.status === "completed" ? pageStyles.chipActive : styles.chipDraft}`}>
                        {sessionStatusLabel(session.status)}
                      </span>
                      <span className={styles.rowMeta}>
                        {session.completed_sets}/{session.total_sets} sets · {formatDuration(session.elapsed_seconds)}
                      </span>
                      {expanded ? <ChevronDown size={16} aria-hidden="true" /> : <ChevronRight size={16} aria-hidden="true" />}
                    </span>
                  </button>

                  {expanded ? (
                    <div className={pageStyles.sessionDetail}>
                      {sessionDetailBusy === session.id ? (
                        <p className={styles.cardText}>Loading session detail…</p>
                      ) : sessionDetailError ? (
                        <p className={styles.error}>{sessionDetailError}</p>
                      ) : detail ? (
                        detail.exercises.length === 0 ? (
                          <p className={styles.cardText}>This session has no exercise snapshots.</p>
                        ) : (
                          detail.exercises.map((exercise) => (
                            <div key={exercise.id} className={pageStyles.exerciseBlock}>
                              <h3 className={pageStyles.exerciseTitle}>{exercise.name}</h3>
                              {exercise.sets.length === 0 ? (
                                <p className={styles.cardText}>No sets were prescribed.</p>
                              ) : (
                                <div className={pageStyles.tableWrap}>
                                  <table className={pageStyles.table}>
                                    <thead>
                                      <tr>
                                        <th scope="col">Set</th>
                                        <th scope="col">Prescribed</th>
                                        <th scope="col">Actual</th>
                                        <th scope="col">Effort</th>
                                        <th scope="col">Status</th>
                                      </tr>
                                    </thead>
                                    <tbody>
                                      {exercise.sets.map((set) => (
                                        <tr key={set.id}>
                                          <td>{set.set_number}</td>
                                          <td>{formatPrescribed(set)}</td>
                                          <td>{formatActual(set)}</td>
                                          <td>{formatEffort(set) || "—"}</td>
                                          <td>
                                            <span className={`${styles.chip} ${set.status === "completed" ? pageStyles.chipActive : styles.chipDraft}`}>
                                              {setStatusLabel(set.status)}
                                            </span>
                                          </td>
                                        </tr>
                                      ))}
                                    </tbody>
                                  </table>
                                </div>
                              )}
                            </div>
                          ))
                        )
                      ) : null}
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        )}

        {historyLoaded && historyTotalPages > 1 ? (
          <div className={styles.pagination}>
            <Button
              variant="secondary"
              size="small"
              disabled={historyPage <= 1 || historyBusy}
              onClick={() => void loadHistory(historyPage - 1)}
            >
              Previous
            </Button>
            <span className={styles.pageInfo}>Page {historyPage} of {historyTotalPages}</span>
            <Button
              variant="secondary"
              size="small"
              disabled={historyPage >= historyTotalPages || historyBusy}
              onClick={() => void loadHistory(historyPage + 1)}
            >
              Next
            </Button>
          </div>
        ) : null}

        {overview.legacy.count > 0 ? (
          <p className={styles.hint}>
            {overview.legacy.count} older completion-only {overview.legacy.count === 1 ? "entry" : "entries"} recorded
            {overview.legacy.last_completed_at ? `, most recently ${formatDate(overview.legacy.last_completed_at)}` : ""}.
            These entries do not include duration or set performance.
          </p>
        ) : null}
      </section>

      <section className={`${styles.sectionGap} ${styles.card}`}>
        <div className={pageStyles.sectionHeading}>
          <div>
            <h2 className={styles.cardTitle}>Exercise history</h2>
            <p className={styles.cardText}>
              Prescribed and actual values are kept separate. Only completed sessions are included.
            </p>
          </div>
        </div>

        {overview.exercises.length === 0 ? (
          <p className={styles.cardText}>Exercise history appears after the client completes a full session.</p>
        ) : (
          <>
            <label className={styles.field}>
              <span className={styles.label}>Exercise</span>
              <select
                className={styles.select}
                value={exerciseId}
                onChange={(event) => void loadExerciseHistory(event.target.value)}
              >
                <option value="">Choose an exercise</option>
                {overview.exercises.map((exercise) => (
                  <option key={exercise.exercise_id} value={exercise.exercise_id}>
                    {exercise.name} · {exercise.session_count} {exercise.session_count === 1 ? "session" : "sessions"}
                  </option>
                ))}
              </select>
            </label>

            {exerciseBusy ? <p className={styles.cardText}>Loading exercise history…</p> : null}
            {exerciseError ? <p className={styles.error}>{exerciseError}</p> : null}

            {!exerciseBusy && !exerciseError && exerciseHistory ? (
              exerciseHistory.sessions.length === 0 ? (
                <p className={styles.cardText}>No completed sessions include this exercise.</p>
              ) : (
                <div className={styles.list}>
                  {exerciseHistory.sessions.map((session) => (
                    <div key={session.session_id} className={styles.card}>
                      <div className={pageStyles.sectionHeading}>
                        <div>
                          <h3 className={styles.cardTitle}>{session.program_name}</h3>
                          <p className={styles.rowMeta}>
                            Week {session.week_number} · workout {session.workout_position} · {formatDateTime(session.completed_at)}
                          </p>
                        </div>
                        <span className={styles.chip}>{session.sets.length} sets</span>
                      </div>
                      <div className={pageStyles.tableWrap}>
                        <table className={pageStyles.table}>
                          <thead>
                            <tr>
                              <th scope="col">Set</th>
                              <th scope="col">Prescribed</th>
                              <th scope="col">Actual</th>
                              <th scope="col">Effort</th>
                              <th scope="col">Status</th>
                            </tr>
                          </thead>
                          <tbody>
                            {session.sets.map((set) => (
                              <tr key={set.id}>
                                <td>{set.set_number}</td>
                                <td>{formatPrescribed(set)}</td>
                                <td>{formatActual(set)}</td>
                                <td>{formatEffort(set) || "—"}</td>
                                <td>
                                  <span className={`${styles.chip} ${set.status === "completed" ? pageStyles.chipActive : styles.chipDraft}`}>
                                    {setStatusLabel(set.status)}
                                  </span>
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  ))}
                </div>
              )
            ) : null}
          </>
        )}
      </section>

      <section className={`${styles.sectionGap} ${pageStyles.mutedNote}`}>
        <Dumbbell size={16} aria-hidden="true" />
        <p>
          This overview reflects recorded RYZE training only. It does not include nutrition, medical data or
          unrecorded activity.
        </p>
      </section>
    </>
  );
};
