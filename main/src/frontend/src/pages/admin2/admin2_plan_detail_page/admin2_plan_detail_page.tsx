import { useCallback, useEffect, useState } from "react";
import { Link, useHistory, useParams } from "react-router-dom";
import { ArrowLeft, Edit, ExternalLink, RefreshCw, Trash2 } from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2StatusBadge } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { Admin2Modal } from "@/components/admin2/admin2_modal/admin2_modal";
import { Button } from "@/components/button/button";
import {
  deleteGenericProgram,
  fetchGenericProgram,
  formatAdminPrice,
  programTypeLabel,
  ProgramStatusEnum,
  publishGenericProgram,
  updateGenericProgram,
  type GenericProgramDetail,
  type GenericProgramSet,
  type ProgramStatusValue
} from "@/services/admin2_api";
import { formatAdminDate } from "@/services/admin_api";
import { detailToPlanDraft, toGenericProgramInput } from "../admin2_plan_create_page/plan_builder_model";

import styles from "./admin2_plan_detail_page.module.css";

const optionalValue = (value: number | null): string => (value === null ? "—" : String(value));

const setCell = (value: string): string => (value.trim() === "" ? "—" : value);

const SetTable = ({ sets }: { sets: GenericProgramSet[] }) => (
  <div className={styles.setTableWrap}>
    <table className={styles.setTable}>
      <thead>
        <tr>
          <th>Set</th>
          <th>Type</th>
          <th>Reps</th>
          <th>Weight</th>
          <th>RIR</th>
          <th>RPE</th>
          <th>Rest</th>
          <th>Tempo</th>
        </tr>
      </thead>
      <tbody>
        {sets.map((set) => (
          <tr key={set.set_number}>
            <td>{set.set_number}</td>
            <td>{setCell(set.set_type)}</td>
            <td>{optionalValue(set.reps)}</td>
            <td>{optionalValue(set.weight_kg)}</td>
            <td>{optionalValue(set.rir)}</td>
            <td>{optionalValue(set.rpe)}</td>
            <td>{optionalValue(set.rest_seconds)}</td>
            <td>{setCell(set.tempo)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

const DeleteDialog = ({
  program,
  onClose,
  onConfirm,
  submitting,
  errorMessage
}: {
  program: GenericProgramDetail;
  onClose: () => void;
  onConfirm: () => void;
  submitting: boolean;
  errorMessage: string;
}) => (
  <Admin2Modal
    title="Delete plan"
    description={`Remove "${program.name}" from the RYZE catalogue.`}
    onClose={onClose}
  >
    <p className={styles.formHint}>
      The plan is soft-deleted: it disappears from the catalogue and the admin list, but its data is
      preserved and can be restored by the platform.
    </p>

    {errorMessage ? (
      <p className={styles.formError} role="alert" style={{ marginTop: "0.5rem" }}>
        {errorMessage}
      </p>
    ) : null}

    <div className={styles.modalActions}>
      <Button type="button" variant="ghost" size="small" onClick={onClose}>
        Cancel
      </Button>
      <Button type="button" variant="danger" size="small" onClick={onConfirm} disabled={submitting}>
        {submitting ? "Deleting…" : "Delete plan"}
      </Button>
    </div>
  </Admin2Modal>
);

export default function Admin2PlanDetailPage() {
  const { programId } = useParams<{ programId: string }>();
  const history = useHistory();
  const [program, setProgram] = useState<GenericProgramDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [showDelete, setShowDelete] = useState(false);
  const [deleteError, setDeleteError] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [publishError, setPublishError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");

    try {
      const result = await fetchGenericProgram(programId);
      setProgram(result);
    } catch {
      setErrorMessage("Unable to load this plan. It may have been removed.");
    } finally {
      setLoading(false);
    }
  }, [programId]);

  useEffect(() => {
    void load();
  }, [load]);

  const confirmDelete = async () => {
    if (!program) {
      return;
    }
    setDeleting(true);
    setDeleteError("");

    try {
      await deleteGenericProgram(program.id);
      history.push("/admin2/plans");
    } catch (error) {
      setDeleteError(
        error instanceof Error ? error.message : "Unable to delete the plan. Please try again."
      );
    } finally {
      setDeleting(false);
    }
  };

  const changeStatus = async (status: ProgramStatusValue) => {
    if (!program) {
      return;
    }
    setPublishing(true);
    setPublishError("");

    try {
      if (status === ProgramStatusEnum.PUBLISHED) {
        await publishGenericProgram(program.id);
      } else {
        const input = toGenericProgramInput({ ...detailToPlanDraft(program), status });
        await updateGenericProgram(program.id, input);
      }
      await load();
    } catch (error) {
      setPublishError(
        error instanceof Error ? error.message : "Unable to update the plan status. Please try again."
      );
    } finally {
      setPublishing(false);
    }
  };

  return (
    <>
      <Admin2PageHeader
        eyebrow="Programs"
        title={program?.name ?? "Plan details"}
        description={program?.description ?? undefined}
        actions={
          <Button
            to="/admin2/plans"
            variant="ghost"
            size="small"
            icon={<ArrowLeft size={15} />}
            iconPosition="left"
          >
            Back to plans
          </Button>
        }
      />

      {loading ? (
        <div className={styles.pageState}>
          <p className={styles.tableMuted}>Loading plan…</p>
        </div>
      ) : errorMessage || !program ? (
        <div className={styles.pageState}>
          <p className={styles.formError}>{errorMessage}</p>
          <div style={{ marginTop: "0.75rem" }}>
            <Button
              variant="secondary"
              size="small"
              onClick={() => void load()}
              icon={<RefreshCw size={15} />}
            >
              Retry
            </Button>
          </div>
        </div>
      ) : (
        <div className={styles.split}>
          <Admin2Section
            title="Program structure"
            subtitle="Weeks, training days, exercises and their set prescriptions"
          >
            <div className={styles.structureBody}>
              {program.weeks.length === 0 ? (
                <p className={styles.structureEmpty}>This plan has no weeks yet.</p>
              ) : (
                program.weeks.map((week) => (
                  <div key={week.week_number} className={styles.weekCard}>
                    <p className={styles.weekTitle}>Week {week.week_number}</p>

                    {week.workouts.length === 0 ? (
                      <p className={styles.structureEmpty}>No training days in this week.</p>
                    ) : (
                      week.workouts.map((workout) => (
                        <div key={workout.position} className={styles.dayCard}>
                          <p className={styles.dayTitle}>Day {workout.position}</p>

                          {workout.exercises.length === 0 ? (
                            <p className={styles.structureEmpty}>No exercises assigned.</p>
                          ) : (
                            workout.exercises.map((exercise) => (
                              <div key={exercise.id} className={styles.exerciseBlock}>
                                <div className={styles.exerciseHead}>
                                  <span className={styles.exercisePos}>{exercise.position}</span>
                                  <span className={styles.exerciseName}>{exercise.name}</span>
                                  <span className={styles.exerciseMeta}>
                                    {exercise.sets.length} set
                                    {exercise.sets.length === 1 ? "" : "s"}
                                  </span>
                                </div>

                                {exercise.instructions ? (
                                  <p className={styles.exerciseNote}>{exercise.instructions}</p>
                                ) : null}
                                {exercise.notes ? (
                                  <p className={styles.exerciseNote}>{exercise.notes}</p>
                                ) : null}

                                <SetTable sets={exercise.sets} />
                              </div>
                            ))
                          )}
                        </div>
                      ))
                    )}
                  </div>
                ))
              )}
            </div>
          </Admin2Section>

          <Admin2Section title="Summary" subtitle="Program facts">
            <div className={styles.detailList}>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Type</p>
                <p className={styles.rowValue}>{programTypeLabel(program.type)}</p>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Status</p>
                <div className={styles.rowValue}>
                  <Admin2StatusBadge
                    label={program.status === ProgramStatusEnum.PUBLISHED ? "Published" : "Draft"}
                    tone={program.status === ProgramStatusEnum.PUBLISHED ? "success" : "warning"}
                  />
                </div>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Level</p>
                <p className={styles.rowValue}>{program.level ?? "—"}</p>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Training type</p>
                <p className={styles.rowValue}>{program.training_type ?? "—"}</p>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Frequency</p>
                <p className={styles.rowValue}>
                  {program.frequency_per_week === null
                    ? "—"
                    : `${program.frequency_per_week} day${program.frequency_per_week === 1 ? "" : "s"}/week`}
                </p>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Duration</p>
                <p className={styles.rowValue}>
                  {program.duration_weeks === null
                    ? "—"
                    : `${program.duration_weeks} week${program.duration_weeks === 1 ? "" : "s"}`}
                </p>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Price</p>
                <p className={styles.rowValue}>
                  {formatAdminPrice(program.price_minor_units, program.currency)}
                </p>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Created</p>
                <p className={styles.rowValue}>{formatAdminDate(program.created_at)}</p>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Last updated</p>
                <p className={styles.rowValue}>{formatAdminDate(program.updated_at)}</p>
              </div>
            </div>

            <div className={styles.actions}>
              {publishError ? (
                <p className={styles.formError} role="alert">
                  {publishError}
                </p>
              ) : null}

              {program.status === ProgramStatusEnum.DRAFT ? (
                <Button
                  variant="secondary"
                  size="small"
                  onClick={() => void changeStatus(ProgramStatusEnum.PUBLISHED)}
                  disabled={publishing}
                >
                  {publishing ? "Publishing…" : "Publish plan"}
                </Button>
              ) : (
                <>
                  <Button to={`/services/generic-program/${program.id}`} variant="secondary" size="small" icon={<ExternalLink size={14} />}>
                    View in marketplace
                  </Button>
                  <Button
                    variant="ghost"
                    size="small"
                    onClick={() => void changeStatus(ProgramStatusEnum.DRAFT)}
                    disabled={publishing}
                  >
                    {publishing ? "Unpublishing…" : "Unpublish plan"}
                  </Button>
                </>
              )}

              <Button
                to={`/admin2/plans/create?edit=${program.id}`}
                variant="secondary"
                size="small"
                icon={<Edit size={14} />}
              >
                Edit plan
              </Button>
              <Button
                variant="ghost"
                size="small"
                icon={<Trash2 size={14} />}
                onClick={() => {
                  setDeleteError("");
                  setShowDelete(true);
                }}
              >
                Delete
              </Button>
              <Link className={styles.backLink} to="/admin2/plans">
                Back to the plan list
              </Link>
            </div>
          </Admin2Section>
        </div>
      )}

      {program && showDelete ? (
        <DeleteDialog
          program={program}
          onClose={() => setShowDelete(false)}
          onConfirm={() => void confirmDelete()}
          submitting={deleting}
          errorMessage={deleteError}
        />
      ) : null}
    </>
  );
}
