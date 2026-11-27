import { ChevronLeft, ChevronRight, ImagePlus, MessageSquarePlus, Quote, Star, X } from "lucide-react";
import type { CSSProperties, FormEvent } from "react";
import { useCallback, useEffect, useId, useRef, useState } from "react";
import { createPortal } from "react-dom";

import { PublicAction } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import {
  FEEDBACK_MESSAGE_MAX,
  FEEDBACK_MESSAGE_MIN,
  feedbackImageUrl,
  fetchFeedbackEligibility,
  fetchPublishedFeedback,
  submitFeedback,
  type FeedbackEligibility,
  type PublishedFeedback
} from "@/services/member_feedback_api";
import { useGettext, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";

import styles from "./community_feedback.module.css";
import { FEEDBACK_PT } from "./feedback_page.pt";

// The gallery opens with COLLAPSED cards and grows a page at a time on
// request ("Show more"), never by endless scrolling (owner request, 2026-11-23).
const PAGE_SIZE = 9;
const COLLAPSED = 6;
const RATINGS = [1, 2, 3, 4, 5] as const;

type FormState =
  | { kind: "closed" }
  | { kind: "loading" }
  | { kind: "signed-out" }
  | { kind: "no-plan" }
  | { kind: "pending" }
  | { kind: "published" }
  | { kind: "form" }
  | { kind: "sent" }
  | { kind: "error" };

const Stars = ({ rating }: { rating: number }) => {
  const tr = useGettext(FEEDBACK_PT);
  return (
  <span className={styles.stars} role="img" aria-label={tr("Rated {rating} out of 5 stars", { rating })}>
    {RATINGS.map((value) => (
      <Star key={value} aria-hidden="true" className={value <= rating ? styles.starOn : styles.starOff} />
    ))}
  </span>
  );
};

// CommunityFeedback asks members for their feedback and lists the approved
// feedback of other members. Submitting requires an account with at least one
// plan (free plans count); the server checks this and publishes a submission
// only after an administrator approves it.
export const CommunityFeedback = () => {
  const tr = useGettext(FEEDBACK_PT);
  const { formatDate } = useI18n();
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
  const [image, setImage] = useState<File | null>(null);
  const [preview, setPreview] = useState("");
  const [expanded, setExpanded] = useState(false);
  // The enlarged card: its index in the loaded list.
  const [openedIndex, setOpenedIndex] = useState<number | null>(null);
  const imageInput = useRef<HTMLInputElement>(null);
  const galleryRef = useRef<HTMLDivElement>(null);
  const messageId = useId();
  const counterId = useId();

  // The local preview of the chosen picture.
  useEffect(() => {
    if (!image) {
      setPreview("");
      return;
    }
    const url = URL.createObjectURL(image);
    setPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [image]);

  const shown = expanded ? items : items.slice(0, COLLAPSED);
  const opened = openedIndex === null ? null : (items[openedIndex] ?? null);
  const step = useCallback(
    (delta: number) => setOpenedIndex((index) => (index === null ? null : (index + delta + items.length) % items.length)),
    [items.length]
  );

  // Escape closes the enlarged card; the arrows move through the gallery.
  useEffect(() => {
    if (openedIndex === null) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpenedIndex(null);
      if (event.key === "ArrowRight") step(1);
      if (event.key === "ArrowLeft") step(-1);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [openedIndex, step]);

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
      setState(
        eligibility.pending
          ? { kind: eligibility.pending.status === "approved" ? "published" : "pending" }
          : eligibility.eligible
            ? { kind: "form" }
            : { kind: "no-plan" }
      );
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
      setFormError(tr("Write at least {count} characters.", { count: FEEDBACK_MESSAGE_MIN }));
      return;
    }
    setSending(true);
    setFormError("");
    try {
      await submitFeedback(rating, message, image);
      setState({ kind: "sent" });
      setImage(null);
    } catch (error) {
      if (error instanceof ApiError && error.code === "FEEDBACK_ALREADY_PENDING") {
        setState({ kind: "pending" });
      } else if (error instanceof ApiError && (error.code === "INVALID_IMAGE" || error.code === "IMAGE_TOO_LARGE" || error.code === "REQUEST_ENTITY_TOO_LARGE")) {
        setFormError(error.code === "INVALID_IMAGE" ? "Use a JPEG or PNG image." : "This image is too large. Try a smaller one.");
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
              <h2 className={ui.title}>{tr("We would love to hear your feedback!")}</h2>
              <p className={ui.body}>{tr("Tell other athletes how your training with RYZE is going.")}</p>
            </div>
            {state.kind === "closed" ? (
              <PublicAction onClick={() => void open()} className={styles.askAction}>
                {tr("Leave feedback")}
              </PublicAction>
            ) : null}
          </div>

          <div aria-live="polite">
            {state.kind === "loading" ? <p className={ui.caption}>{tr("Checking your account…")}</p> : null}

            {state.kind === "signed-out" ? (
              <div className={styles.notice}>
                <p className={ui.body}>{tr("Log in to share your feedback. Any plan on your account counts, free ones too.")}</p>
                <div className={ui.actions}>
                  <PublicAction to="/login">{tr("Log in")}</PublicAction>
                  <PublicAction to="/register" variant="ghostOnDark">
                    {tr("Create account")}
                  </PublicAction>
                </div>
              </div>
            ) : null}

            {state.kind === "no-plan" ? (
              <div className={styles.notice}>
                <p className={ui.body}>{tr("Feedback opens once you have a plan. Free training plans count too.")}</p>
                <PublicAction to="/services/generic-program" variant="ghostOnDark">
                  {tr("Find a plan")}
                </PublicAction>
              </div>
            ) : null}

            {state.kind === "pending" ? (
              <p className={joinClassNames(ui.body, styles.notice)}>
                {tr("Thanks! Your feedback is being reviewed. Each member leaves one feedback.")}
              </p>
            ) : null}

            {state.kind === "published" ? (
              <p className={joinClassNames(ui.body, styles.notice, styles.success)}>
                {tr("You have already left your feedback — it is in the gallery. Thank you!")}
              </p>
            ) : null}

            {state.kind === "sent" ? (
              <p className={joinClassNames(ui.body, styles.notice, styles.success)}>
                {tr("Thank you! Your feedback will appear here after a quick review.")}
              </p>
            ) : null}

            {state.kind === "error" ? (
              <div className={styles.notice}>
                <p className={ui.body}>{tr("We couldn't check your account right now.")}</p>
                <PublicAction onClick={() => void open()} variant="ghostOnDark">
                  {tr("Try again")}
                </PublicAction>
              </div>
            ) : null}
          </div>

          {state.kind === "form" ? (
            <form className={styles.form} onSubmit={handleSubmit} noValidate>
              <fieldset className={styles.rating}>
                <legend className={styles.legend}>{tr("Your rating")}</legend>
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
                      <span className={ui.srOnly}>{tr(value === 1 ? "{count} star" : "{count} stars", { count: value })}</span>
                    </label>
                  ))}
                </div>
              </fieldset>

              <label className={styles.field} htmlFor={messageId}>
                <span className={styles.legend}>{tr("Your feedback")}</span>
                <textarea
                  id={messageId}
                  value={message}
                  maxLength={FEEDBACK_MESSAGE_MAX}
                  rows={5}
                  placeholder={tr("What changed for you since you started?")}
                  aria-describedby={counterId}
                  onChange={(event) => setMessage(event.target.value)}
                />
                <span id={counterId} className={ui.caption}>
                  {tr("{length}/{max} · at least {min} characters. Shown with your first name and last initial.", {
                    length: trimmedLength,
                    max: FEEDBACK_MESSAGE_MAX,
                    min: FEEDBACK_MESSAGE_MIN
                  })}
                </span>
              </label>

              <div className={styles.field}>
                <span className={styles.legend}>{tr("Your photo (optional)")}</span>
                <input
                  ref={imageInput}
                  type="file"
                  accept="image/jpeg,image/png"
                  className={ui.srOnly}
                  tabIndex={-1}
                  aria-hidden="true"
                  onChange={(event) => {
                    setImage(event.target.files?.[0] ?? null);
                    event.target.value = "";
                  }}
                />
                {preview ? (
                  <div className={styles.imagePreview}>
                    <img src={preview} alt={tr("Your chosen photo")} />
                    <button type="button" className={styles.imageRemove} onClick={() => setImage(null)} aria-label={tr("Remove photo")}>
                      <X aria-hidden="true" />
                    </button>
                  </div>
                ) : (
                  <button type="button" className={styles.imagePick} onClick={() => imageInput.current?.click()}>
                    <ImagePlus aria-hidden="true" />
                    <span>{tr("Add one photo — your progress, your gym, a workout. JPEG or PNG.")}</span>
                  </button>
                )}
              </div>

              {formError ? (
                <p className={styles.error} role="alert">
                  {tr(formError)}
                </p>
              ) : null}

              <div className={ui.actions}>
                <PublicAction type="submit" disabled={!canSend}>
                  {sending ? tr("Sending…") : tr("Send feedback")}
                </PublicAction>
                <PublicAction variant="ghostOnDark" onClick={() => setState({ kind: "closed" })}>
                  {tr("Cancel")}
                </PublicAction>
              </div>
            </form>
          ) : null}
        </div>

        <div className={styles.community}>
          <h2 id="community-heading" className={ui.heading}>
            {tr("Other feedback from")} <span className={ui.hl}>{tr("our athletes.")}</span>
          </h2>

          {listState === "error" && items.length === 0 ? (
            <div className={styles.notice}>
              <p className={ui.body}>{tr("Feedback couldn't be loaded right now.")}</p>
              <PublicAction variant="ghostOnDark" onClick={() => void loadPage(1)}>
                {tr("Try again")}
              </PublicAction>
            </div>
          ) : listState === "ready" && items.length === 0 ? (
            <p className={ui.body}>{tr("No feedback has been published yet. Be the first to share yours.")}</p>
          ) : (
            <div ref={galleryRef} className={styles.galleryWrap}>
              <ul className={styles.gallery}>
                {shown.map((item, index) => (
                  <li
                    key={item.id}
                    className={joinClassNames(styles.galleryItem, index === 0 && item.has_image && styles.featured, !item.has_image && styles.textOnly)}
                    style={{ "--i": index } as CSSProperties}
                  >
                    <button
                      type="button"
                      className={joinClassNames(ui.surface, styles.card)}
                      onClick={() => setOpenedIndex(index)}
                      aria-label={tr("Open feedback from {name}", { name: item.display_name })}
                    >
                      {item.has_image ? (
                        <span className={styles.cardMedia}>
                          <img className={styles.cardImage} src={feedbackImageUrl(item.id)} alt="" loading="lazy" decoding="async" />
                          <span className={styles.cardBadge}>
                            <Stars rating={item.rating} />
                          </span>
                        </span>
                      ) : null}
                      <span className={styles.cardBody}>
                        {item.has_image ? null : (
                          <span className={styles.cardQuoteHead}>
                            <Quote aria-hidden="true" className={styles.quoteMark} />
                            <Stars rating={item.rating} />
                          </span>
                        )}
                        <span className={styles.cardMessage}>{item.message}</span>
                        <span className={styles.cardAuthor}>
                          <span className={styles.avatar} aria-hidden="true">
                            {item.display_name.charAt(0).toUpperCase()}
                          </span>
                          <span>
                            <strong>{item.display_name}</strong>
                            <span className={ui.caption}>{formatDate(item.published_at)}</span>
                          </span>
                        </span>
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {listState === "loading" ? <p className={ui.caption}>{tr("Loading feedback…")}</p> : null}
          <div className={styles.galleryActions}>
            {listState === "ready" && (!expanded ? items.length > COLLAPSED || page < totalPages : page < totalPages) ? (
              <PublicAction
                variant="ghostOnDark"
                onClick={() => {
                  if (!expanded) {
                    setExpanded(true);
                    if (items.length <= COLLAPSED && page < totalPages) void loadPage(page + 1);
                  } else {
                    void loadPage(page + 1);
                  }
                }}
              >
                {tr("Show more")}
              </PublicAction>
            ) : null}
            {expanded ? (
              <PublicAction
                variant="ghostOnDark"
                onClick={() => {
                  setExpanded(false);
                  galleryRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
                }}
              >
                {tr("Show less")}
              </PublicAction>
            ) : null}
          </div>
        </div>
      </div>

      {opened && openedIndex !== null
        ? createPortal(
            <div className={styles.lightbox} role="dialog" aria-modal="true" aria-label={tr("Feedback from {name}", { name: opened.display_name })} onClick={() => setOpenedIndex(null)}>
              <figure className={joinClassNames(styles.lightboxCard, !opened.has_image && styles.lightboxTextOnly)} onClick={(event) => event.stopPropagation()}>
                <button type="button" className={styles.lightboxClose} onClick={() => setOpenedIndex(null)} aria-label={tr("Close")} autoFocus>
                  <X aria-hidden="true" />
                </button>
                {opened.has_image ? (
                  <div className={styles.lightboxMedia}>
                    <img className={styles.lightboxImage} src={feedbackImageUrl(opened.id)} alt="" />
                  </div>
                ) : null}
                <figcaption className={styles.lightboxBody}>
                  <Quote aria-hidden="true" className={styles.quoteMark} />
                  <Stars rating={opened.rating} />
                  <p className={styles.lightboxMessage}>{opened.message}</p>
                  <span className={styles.cardAuthor}>
                    <span className={styles.avatar} aria-hidden="true">
                      {opened.display_name.charAt(0).toUpperCase()}
                    </span>
                    <span>
                      <strong>{opened.display_name}</strong>
                      <span className={styles.lightboxDate}>{formatDate(opened.published_at)}</span>
                    </span>
                  </span>
                  {items.length > 1 ? (
                    <div className={styles.lightboxNav}>
                      <button type="button" onClick={() => step(-1)} aria-label={tr("Previous")}>
                        <ChevronLeft aria-hidden="true" />
                      </button>
                      <span>
                        {openedIndex + 1} / {items.length}
                      </span>
                      <button type="button" onClick={() => step(1)} aria-label={tr("Next")}>
                        <ChevronRight aria-hidden="true" />
                      </button>
                    </div>
                  ) : null}
                </figcaption>
              </figure>
            </div>,
            document.body
          )
        : null}
    </section>
  );
};
