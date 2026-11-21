import { ArrowLeft } from "lucide-react";
import { Link, useHistory } from "react-router-dom";

import { useGoBack } from "@/hooks/use_go_back";
import { useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./back_button.module.css";

interface BackButtonProps {
  // Where to go when there is no earlier RYZE page in this tab.
  fallback?: string | (() => string);
  // The page "above" this one: when given, the button always goes there (one
  // click from any depth) instead of one step back in the history.
  to?: string;
  // With `to`: that page is the previous one of this tab, so the button steps
  // back in the history (the page is restored where it was left).
  viaHistory?: boolean;
  // "icon": a round button (navbar); "pill": arrow and label.
  variant?: "icon" | "pill";
  label?: string;
  className?: string;
}

// BackButton goes back one page (owner request, 2026-11-15), or, with `to`,
// up to the named parent page (owner request, 2026-11-20: from anywhere in the
// personal area one click returns to it, never five steps of history).
export const BackButton = ({ fallback = "/", to, viaHistory = false, variant = "pill", label, className }: BackButtonProps) => {
  const { locale } = useI18n();
  const history = useHistory();
  const goBack = useGoBack(fallback);
  const text = label ?? (locale === "pt" ? "Voltar" : "Back");
  if (to && !viaHistory) {
    return (
      <Link
        to={to}
        className={joinClassNames(styles.back, variant === "icon" ? styles.icon : styles.pill, className)}
        aria-label={text}
        title={text}
      >
        <ArrowLeft aria-hidden="true" />
        {variant === "icon" ? null : <span>{text}</span>}
      </Link>
    );
  }
  return (
    <button
      type="button"
      className={joinClassNames(styles.back, variant === "icon" ? styles.icon : styles.pill, className)}
      onClick={to && viaHistory ? () => history.goBack() : goBack}
      // Always named: the label may be hidden on phones (arrow only).
      aria-label={text}
      title={text}
    >
      <ArrowLeft aria-hidden="true" />
      {variant === "icon" ? null : <span>{text}</span>}
    </button>
  );
};
