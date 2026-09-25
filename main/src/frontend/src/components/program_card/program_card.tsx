import { Link } from "react-router-dom";

import { BRAND_ASSETS } from "@/constants/brand_assets";
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
}

const part = (value: string | null): string => value ?? "";

// Combines non-null generic metadata into a single muted line, e.g.
// "12 weeks · 4 days/week". Empty lines are omitted.
const buildMetaLine = (items: string[]): string => items.filter(Boolean).join(" · ");

export const ProgramCard = ({ program, forceFree = false, owned = false }: ProgramCardProps) => {
  const schedule = buildMetaLine([
    program.duration_weeks !== null ? `${program.duration_weeks} weeks` : "",
    program.frequency_per_week !== null ? `${program.frequency_per_week} days/week` : ""
  ]);
  const category = buildMetaLine([part(program.training_type), part(program.level)]);
  const isFree = forceFree || program.type === FREE_PROGRAM_TYPE;
  const price = formatMarketplacePrice(
    forceFree ? 0 : program.price_minor_units,
    program.currency,
    program.type
  );

  return (
    <article className={styles.card}>
      <Link
        to={owned ? `/services/my-programs/${program.id}` : `/services/generic-program/${program.id}`}
        className={styles.link}
        aria-label={owned ? `Open ${program.name}` : `View ${program.name}`}
      >
        <div className={styles.image} aria-hidden="true">
          <img className={styles.logo} src={BRAND_ASSETS.icon} alt="" />
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
            <span className={styles.owned}>Owned</span>
          ) : (
            <span className={isFree ? styles.priceFree : styles.price}>{price}</span>
          )}
        </div>
      </Link>
    </article>
  );
};