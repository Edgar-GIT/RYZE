import { Link } from "react-router-dom";

import { BRAND_ASSETS } from "@/constants/brand_assets";
import { useMarketplaceCopy } from "@/constants/marketplace_copy";
import { coachInitials, type PublicCoach } from "@/services/coaches_api";
import {
  formatMarketplacePrice,
  FREE_PROGRAM_TYPE,
  type MarketplaceProgram
} from "@/services/marketplace_api";

import styles from "./program_card.module.css";

interface ProgramCardProps {
  program: MarketplaceProgram;
  forceFree?: boolean;
  owned?: boolean;
  // The coach who created the program (coach showcase). Shown under the card
  // as a separate link to the coach profile.
  coach?: PublicCoach | null;
}

// Each card gets one of four brand gradients, chosen from its id so a card
// keeps its colour wherever it appears.
const TONES = ["royal", "ring", "cyan", "deep"] as const;
const toneFor = (id: string) => TONES[[...id].reduce((sum, character) => sum + character.charCodeAt(0), 0) % TONES.length];

// Combines non-null generic metadata into a single muted line, e.g.
// "12 weeks · 4 days/week". Empty lines are omitted.
const buildMetaLine = (items: string[]): string => items.filter(Boolean).join(" · ");

export const ProgramCard = ({ program, forceFree = false, owned = false, coach = null }: ProgramCardProps) => {
  const { text, locale, valueLabel } = useMarketplaceCopy();
  const schedule = buildMetaLine([
    program.duration_weeks !== null ? text.weeks(program.duration_weeks) : "",
    program.frequency_per_week !== null ? text.daysPerWeek(program.frequency_per_week) : ""
  ]);
  const category = buildMetaLine([valueLabel(program.training_type), valueLabel(program.level)]);
  const isFree = forceFree || program.type === FREE_PROGRAM_TYPE;
  const price = formatMarketplacePrice(
    forceFree ? 0 : program.price_minor_units,
    program.currency,
    program.type,
    locale
  );

  return (
    <article className={styles.card}>
      <Link
        to={owned ? `/services/my-programs/${program.id}` : `/services/generic-program/${program.id}`}
        className={styles.link}
        aria-label={owned ? text.open(program.name) : text.view(program.name)}
      >
        <div className={styles.image} data-tone={toneFor(program.id)} aria-hidden="true">
          <img className={styles.logo} src={BRAND_ASSETS.icon} alt="" />
          {isFree && !owned ? <span className={styles.freeBadge}>{text.free}</span> : null}
        </div>
        <div className={styles.body}>
          <h3 className={styles.title}>{program.name}</h3>
          {schedule || category ? (
            <div className={styles.meta}>
              {schedule ? <span className={styles.metaItem}>{schedule}</span> : null}
              {category ? <span className={styles.metaItem}>{category}</span> : null}
            </div>
          ) : null}
          {owned ? (
            <span className={styles.owned}>{text.owned}</span>
          ) : (
            <span className={isFree ? styles.priceFree : styles.price}>{price}</span>
          )}
        </div>
      </Link>
      {coach ? (
        <Link to={`/coaches/${encodeURIComponent(coach.id)}`} className={styles.coach} aria-label={text.coachLink(coach.name)}>
          <span className={styles.coachAvatar} aria-hidden="true">
            {coachInitials(coach.name)}
          </span>
          <span className={styles.coachName}>{coach.name}</span>
        </Link>
      ) : null}
    </article>
  );
};