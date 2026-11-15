import { ArrowLeft } from "lucide-react";

import { useGoBack } from "@/hooks/use_go_back";
import { useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./back_button.module.css";

interface BackButtonProps {
  // Where to go when there is no earlier RYZE page in this tab.
  fallback: string | (() => string);
  // "icon": a round button (navbar); "pill": arrow and label.
  variant?: "icon" | "pill";
  label?: string;
  className?: string;
}

// BackButton goes back one page (owner request, 2026-11-15).
export const BackButton = ({ fallback, variant = "pill", label, className }: BackButtonProps) => {
  const { locale } = useI18n();
  const goBack = useGoBack(fallback);
  const text = label ?? (locale === "pt" ? "Voltar" : "Back");
  return (
    <button
      type="button"
      className={joinClassNames(styles.back, variant === "icon" ? styles.icon : styles.pill, className)}
      onClick={goBack}
      // Always named: the label may be hidden on phones (arrow only).
      aria-label={text}
      title={text}
    >
      <ArrowLeft aria-hidden="true" />
      {variant === "icon" ? null : <span>{text}</span>}
    </button>
  );
};
