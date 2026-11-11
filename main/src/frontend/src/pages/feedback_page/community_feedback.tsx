import { MessageSquarePlus, Star } from "lucide-react";
import type { FormEvent } from "react";
import { useCallback, useEffect, useId, useState } from "react";

import { PublicAction } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import {
  FEEDBACK_MESSAGE_MAX,
  FEEDBACK_MESSAGE_MIN,
  fetchFeedbackEligibility,
  fetchPublishedFeedback,
  submitFeedback,
  type FeedbackEligibility,
  type PublishedFeedback
} from "@/services/member_feedback_api";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";

import styles from "./community_feedback.module.css";

const PAGE_SIZE = 6;
const RATINGS = [1, 2, 3, 4, 5] as const;

type FormState =
  | { kind: "closed" }
  | { kind: "loading" }
  | { kind: "signed-out" }
  | { kind: "no-plan" }
  | { kind: "pending" }
  | { kind: "form" }
  | { kind: "sent" }
  | { kind: "error" };

const formatDate = (value: string) =>
  new Date(value).toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" });

const Stars = ({ rating }: { rating: number }) => (
  <span className={styles.stars} role="img" aria-label={`Rated ${rating} out of 5 stars`}>
    {RATINGS.map((value) => (
      <Star key={value} aria-hidden="true" className={value <= rating ? styles.starOn : styles.starOff} />
    ))}
  </span>
);

// CommunityFeedback asks members for their feedback and lists the approved
// feedback of other members. Submitting requires an account with at least one
// plan (free plans count); the server checks this and publishes a submission
// only after an administrator approves it.
export const CommunityFeedback = () => {
  const [state, setState] = useState<FormState>({ kind: "closed" });
  const [rating, setRating] = useState(0);
  const [hovered, setHovered] = useState(0);
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);
  const [formError, setFormError] = useState("");
  const [items, setItems] = useState<PublishedFeedback[]>([]);
  const [page, setPage] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [listState, setListState] = useState<"loading" | "ready" | "error">("loading");
  const messageId = useId();
  const counterId = useId();

  const loadPage = useCallback(async (next: number) => {
    setListState("loading");
    try {
      const result = await fetchPublishedFeedback(next, PAGE_SIZE);
      setItems((current) => (next === 1 ? result.feedback : [...current, ...result.feedback]));
      setPage(result.pagination.page);
      setTotalPages(result.pagination.total_pages);
      setListState("ready");
    } catch {
      setListState("error");
    }
  }, []);

  useEffect(() => {
    void loadPage(1);
  }, [loadPage]);

  const open = async () => {
    setState({ kind: "loading" });
    try {
      const eligibility: FeedbackEligibility = await fetchFeedbackEligibility();
      setState(eligibility.pending ? { kind: "pending" } : eligibility.eligible ? { kind: "form" } : { kind: "no-plan" });
    } catch (error) {
      setState(error instanceof ApiError && error.status === 401 ? { kind: "signed-out" } : { kind: "error" });
    }
  };

  const trimmedLength = message.trim().length;
  const canSend = rating > 0 && trimmedLength >= FEEDBACK_MESSAGE_MIN && trimmedLength <= FEEDBACK_MESSAGE_MAX && !sending;

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (rating === 0) {
      setFormError("Choose a rating from 1 to 5 stars.");
      return;
    }
    if (trimmedLength < FEEDBACK_MESSAGE_MIN) {
      setFormError(`Write at least ${FEEDBACK_MESSAGE_MIN} characters.`);
      return;
    }
    setSending(true);
    setFormError("");
    try {
      await submitFeedback(rating, message);
      setState({ kind: "sent" });
    } catch (error) {
      if (error instanceof ApiError && error.code === "FEEDBACK_ALREADY_PENDING") {
        setState({ kind: "pending" });
      } else if (error instanceof ApiError && error.code === "FEEDBACK_NOT_ELIGIBLE") {
        setState({ kind: "no-plan" });
      } else if (error instanceof ApiError && error.status === 401) {
        setState({ kind: "signed-out" });
      } else {
        setFormError("Your feedback could not be sent. Check it and try again.");
      }
    } finally {
      setSending(false);
    }
  };

  return (
    <section className={joinClassNames(ui.section, ui.toneNavy, styles.section)} aria-labelledby="community-heading">
      <div className={joinClassNames(ui.container, styles.layout)}>
        <div className={joinClassNames(ui.surface, styles.ask)}>
          <div className={styles.askHead}>
            <span className={ui.iconTile} aria-hidden="true">
              <MessageSquarePlus strokeWidth={1.9} />
            </span>
            <div>
              <h2 className={ui.title}>We would love to hear your feedback!</h2>
              <p className={ui.body}>Tell other athletes how your training with RYZE is going.</p>
            </div>
            {state.kind === "closed" ? (
              <PublicAction onClick={() => void open()} className={styles.askAction}>
                Leave feedback
              </PublicAction>
            ) : null}
          </div>

          <div aria-live="polite">
            {state.kind === "loading" ? <p className={ui.caption}>Checking your account…</p> : null}

            {state.kind === "signed-out" ? (
              <div className={styles.notice}>
                <p className={ui.body}>Log in to share your feedback. Any plan on your account counts, free ones too.</p>
                <div className={ui.actions}>
                  <PublicAction to="/login">Log in</PublicAction>
                  <PublicAction to="/register" variant="ghostOnDark">
                    Create account
                  </PublicAction>
                </div>
              </div>
            ) : null}

            {state.kind === "no-plan" ? (
              <div className={styles.notice}>
                <p className={ui.body}>Feedback opens once you have a plan. Free training plans count too.</p>
                <PublicAction to="/services/generic-program" variant="ghostOnDark">
                  Find a plan
                </PublicAction>
              </div>
            ) : null}

            {state.kind === "pending" ? (
              <p className={joinClassNames(ui.body, styles.notice)}>
                Thanks! Your feedback is being reviewed. You can send another once it has been reviewed.
              </p>
            ) : null}

            {state.kind === "sent" ? (
              <p className={joinClassNames(ui.body, styles.notice, styles.success)}>
                Thank you! Your feedback will appear here after a quick review.
              </p>
            ) : null}

            {state.kind === "error" ? (
              <div className={styles.notice}>
                <p className={ui.body}>We couldn&apos;t check your account right now.</p>
                <PublicAction onClick={() => void open()} variant="ghostOnDark">
                  Try again
                </PublicAction>
              </div>
            ) : null}
          </div>

          {state.kind === "form" ? (
            <form className={styles.form} onSubmit={handleSubmit} noValidate>
              <fieldset className={styles.rating}>
                <legend className={styles.legend}>Your rating</legend>
                <div className={styles.starPicker} onMouseLeave={() => setHovered(0)}>
                  {RATINGS.map((value) => (
                    <label
                      key={value}
                      className={joinClassNames(styles.starOption, value <= (hovered || rating) && styles.starOptionOn)}
                      onMouseEnter={() => setHovered(value)}
                    >
                      <input
                        type="radio"
                        name="rating"
                        value={value}
                        checked={rating === value}
                        onChange={() => setRating(value)}
                        className={ui.srOnly}
                      />
                      <Star aria-hidden="true" />
                      <span className={ui.srOnly}>
                        {value} {value === 1 ? "star" : "stars"}
                      </span>
                    </label>
                  ))}
                </div>
              </fieldset>

              <label className={styles.field} htmlFor={messageId}>
                <span className={styles.legend}>Your feedback</span>
                <textarea
                  id={messageId}
                  value={message}
                  maxLength={FEEDBACK_MESSAGE_MAX}
                  rows={5}
                  placeholder="What changed for you since you started?"
                  aria-describedby={counterId}
                  onChange={(event) => setMessage(event.target.value)}
                />
                <span id={counterId} className={ui.caption}>
                  {trimmedLength}/{FEEDBACK_MESSAGE_MAX} · at least {FEEDBACK_MESSAGE_MIN} characters. Shown with your
                  first name and last initial.
                </span>
              </label>

              {formError ? (
                <p className={styles.error} role="alert">
                  {formError}
                </p>
              ) : null}

              <div className={ui.actions}>
                <PublicAction type="submit" disabled={!canSend}>
                  {sending ? "Sending…" : "Send feedback"}
                </PublicAction>
                <PublicAction variant="ghostOnDark" onClick={() => setState({ kind: "closed" })}>
                  Cancel
                </PublicAction>
              </div>
            </form>
          ) : null}
        </div>

        <div className={styles.community}>
          <h2 id="community-heading" className={ui.heading}>
            Other feedback from <span className={ui.hl}>our athletes.</span>
          </h2>

          {listState === "error" && items.length === 0 ? (
            <div className={styles.notice}>
              <p className={ui.body}>Feedback couldn&apos;t be loaded right now.</p>
              <PublicAction variant="ghostOnDark" onClick={() => void loadPage(1)}>
                Try again
              </PublicAction>
            </div>
          ) : listState === "ready" && items.length === 0 ? (
            <p className={ui.body}>No feedback has been published yet. Be the first to share yours.</p>
          ) : (
            <ul className={styles.list}>
              {items.map((item) => (
                <li key={item.id} className={joinClassNames(ui.surface, styles.item)}>
                  <div className={styles.itemHead}>
                    <span className={styles.avatar} aria-hidden="true">
                      {item.display_name.charAt(0).toUpperCase()}
                    </span>
                    <div>
                      <strong>{item.display_name}</strong>
                      <span className={ui.caption}>{formatDate(item.published_at)}</span>
                    </div>
                    <Stars rating={item.rating} />
                  </div>
                  <p className={styles.message}>{item.message}</p>
                </li>
              ))}
            </ul>
          )}

          {listState === "loading" ? <p className={ui.caption}>Loading feedback…</p> : null}
          {listState === "ready" && page < totalPages ? (
            <PublicAction variant="ghostOnDark" onClick={() => void loadPage(page + 1)} className={styles.more}>
              Show more
            </PublicAction>
          ) : null}
        </div>
      </div>
    </section>
  );
};
