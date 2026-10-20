import { ArrowLeft } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useHistory, useParams } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { Exercise3DDemo } from "@/components/exercise_library/exercise_3d_demo";
import { ExercisePreviewBody } from "@/components/exercise_library/exercise_preview_body";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { fetchExerciseDetail, type ExerciseDetail } from "@/services/exercise_library";
import { ApiError } from "@utils/http_client";

import styles from "./exercise_detail_page.module.css";

export const ExerciseDetailPage = () => {
  const { exerciseId } = useParams<{ exerciseId: string }>();
  const history = useHistory();

  const [detail, setDetail] = useState<ExerciseDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [error, setError] = useState("");
  const [reloadToken, setReloadToken] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    setNotFound(false);
    setDetail(null);

    fetchExerciseDetail(exerciseId)
      .then((result) => {
        if (!cancelled) {
          setDetail(result);
        }
      })
      .catch((reason: unknown) => {
        if (cancelled) {
          return;
        }
        if (reason instanceof ApiError && reason.code === "EXERCISE_NOT_FOUND") {
          setNotFound(true);
        } else {
          setError("Unable to load this exercise. Please try again.");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    window.scrollTo({ top: 0 });

    return () => {
      cancelled = true;
    };
  }, [exerciseId, reloadToken]);

  return (
    <PageWrapper className={styles.page}>
      <section className={styles.detail}>
        <Container size="default" className={styles.container}>
          <Link to="/exercises" className={styles.backLink}>
            <ArrowLeft size={15} aria-hidden="true" />
            Exercise Library
          </Link>

          {loading ? (
            <div className={styles.pageState}>
              <p>Loading exercise…</p>
            </div>
          ) : notFound ? (
            <div className={styles.pageState}>
              <p>This exercise does not exist or is no longer available.</p>
              <Button variant="secondary" size="small" to="/exercises">
                Browse the Exercise Library
              </Button>
            </div>
          ) : error || !detail ? (
            <div className={styles.pageState}>
              <p>{error || "Unable to load this exercise."}</p>
              <Button
                variant="secondary"
                size="small"
                onClick={() => setReloadToken((current) => current + 1)}
              >
                Retry
              </Button>
            </div>
          ) : (
            <article className={styles.card}>
              <ExercisePreviewBody
                exercise={detail.exercise}
                alternatives={detail.alternatives}
                onSelectAlternative={(alternativeId) => history.push(`/exercises/${alternativeId}`)}
                titleAs="h1"
              />
              <Exercise3DDemo
                key={detail.exercise.id}
                threeDKey={detail.exercise.three_d_key}
                exerciseName={detail.exercise.name}
              />
            </article>
          )}
        </Container>
      </section>
    </PageWrapper>
  );
};
