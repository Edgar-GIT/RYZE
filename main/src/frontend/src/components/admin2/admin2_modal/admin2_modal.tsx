import type { ReactNode } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./admin2_modal.module.css";

interface Admin2ModalProps {
  title: string;
  description?: string;
  onClose: () => void;
  children: ReactNode;
  className?: string;
}

export const Admin2Modal = ({ title, description, onClose, children, className }: Admin2ModalProps) => (
  <div
    className={styles.overlay}
    role="dialog"
    aria-modal="true"
    aria-label={title}
    onClick={onClose}
    onKeyDown={(event) => {
      if (event.key === "Escape") {
        onClose();
      }
    }}
  >
    <div
      className={joinClassNames(styles.card, className)}
      role="document"
      onClick={(event) => event.stopPropagation()}
    >
      <header className={styles.header}>
        <h2 className={styles.title}>{title}</h2>
        {description ? <p className={styles.description}>{description}</p> : null}
      </header>
      {children}
    </div>
  </div>
);