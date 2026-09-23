import { useCallback, useEffect, useRef, useState } from "react";

import { CarouselArrow } from "@/components/carousel_arrow/carousel_arrow";
import { ProgramCard } from "@/components/program_card/program_card";
import type { MarketplaceProgram } from "@/services/marketplace_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./program_row.module.css";

const PAGE_SIZE = 3;
const SLIDE_DURATION_MS = 400;

interface ProgramRowProps {
  title: string;
  description?: string;
  programs: MarketplaceProgram[];
}

type Direction = "back" | "forward";
type Phase = "idle" | "exit" | "enter";

export const ProgramRow = ({ title, description, programs }: ProgramRowProps) => {
  const [page, setPage] = useState(0);
  const [direction, setDirection] = useState<Direction>("forward");
  const [phase, setPhase] = useState<Phase>("idle");
  const timersRef = useRef<number[]>([]);

  const totalPages = Math.ceil(programs.length / PAGE_SIZE);
  const visible = programs.slice(page * PAGE_SIZE, page * PAGE_SIZE + PAGE_SIZE);
  const slots = Array.from({ length: PAGE_SIZE }, (_, index) => visible[index] ?? null);

  const atStart = page <= 0;
  const atEnd = page >= totalPages - 1;

  useEffect(() => {
    const timers = timersRef.current;
    return () => timers.forEach((timer) => window.clearTimeout(timer));
  }, []);

  const travel = useCallback((nextDirection: Direction, nextPage: number) => {
    timersRef.current.forEach((timer) => window.clearTimeout(timer));
    timersRef.current = [];

    setDirection(nextDirection);
    setPhase("exit");

    const swap = window.setTimeout(() => {
      setPage(nextPage);
      setPhase("enter");
    }, SLIDE_DURATION_MS);
    const settle = window.setTimeout(() => setPhase("idle"), SLIDE_DURATION_MS * 2);

    timersRef.current.push(swap, settle);
  }, []);

  const goNext = useCallback(() => {
    if (!atEnd) travel("forward", page + 1);
  }, [atEnd, page, travel]);

  const goBack = useCallback(() => {
    if (!atStart) travel("back", page - 1);
  }, [atStart, page, travel]);

  const slideClass = (() => {
    if (phase === "exit") {
      return direction === "forward"
        ? styles.slideExitForward
        : styles.slideExitBack;
    }
    if (phase === "enter") {
      return direction === "forward"
        ? styles.slideEnterForward
        : styles.slideEnterBack;
    }
    return "";
  })();

  return (
    <section className={styles.row} aria-label={title}>
      <header className={styles.header}>
        <h2 className={styles.title}>{title}</h2>
        {description ? <p className={styles.description}>{description}</p> : null}
      </header>

      <div className={styles.grid}>
        <CarouselArrow
          direction="back"
          onClick={goBack}
          disabled={atStart}
          ariaLabel={`Previous ${title} programs`}
        />

        <div className={styles.cards}>
          <div className={joinClassNames(styles.cardRow, slideClass)}>
            {slots.map((program, index) => (
              <div key={program ? program.id : `slot-${index}`} className={styles.slot}>
                {program ? (
                  <ProgramCard program={program} />
                ) : (
                  <div className={styles.placeholder} aria-hidden="true" />
                )}
              </div>
            ))}
          </div>
        </div>

        <CarouselArrow
          direction="forward"
          onClick={goNext}
          disabled={atEnd}
          ariaLabel={`Next ${title} programs`}
        />
      </div>
    </section>
  );
};