import { Link } from "react-router-dom";
import type { ReactNode } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./button.module.css";

type ButtonVariant = "primary" | "secondary" | "ghost" | "dark";
type ButtonSize = "small" | "medium" | "large";
type ButtonType = "button" | "submit" | "reset";

interface ButtonProps {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
  to?: string;
  href?: string;
  type?: ButtonType;
  disabled?: boolean;
  icon?: ReactNode;
  iconPosition?: "left" | "right";
  ariaLabel?: string;
  className?: string;
  onClick?: () => void;
}

export const Button = ({
  children,
  variant = "primary",
  size = "medium",
  to,
  href,
  type = "button",
  disabled = false,
  icon,
  iconPosition = "right",
  ariaLabel,
  className,
  onClick
}: ButtonProps) => {
  const buttonClassName = joinClassNames(
    styles.button,
    styles[variant],
    styles[size],
    disabled && styles.disabled,
    className
  );

  const content = (
    <>
      {icon && iconPosition === "left" ? (
        <span className={styles.icon} aria-hidden="true">
          {icon}
        </span>
      ) : null}
      <span>{children}</span>
      {icon && iconPosition === "right" ? (
        <span className={styles.icon} aria-hidden="true">
          {icon}
        </span>
      ) : null}
    </>
  );

  if (to && !disabled) {
    return (
      <Link to={to} className={buttonClassName} aria-label={ariaLabel}>
        {content}
      </Link>
    );
  }

  if (href && !disabled) {
    const isExternal = href.startsWith("http");

    return (
      <a
        className={buttonClassName}
        href={href}
        aria-label={ariaLabel}
        target={isExternal ? "_blank" : undefined}
        rel={isExternal ? "noreferrer" : undefined}
      >
        {content}
      </a>
    );
  }

  return (
    <button
      className={buttonClassName}
      type={type}
      disabled={disabled}
      aria-label={ariaLabel}
      onClick={onClick}
    >
      {content}
    </button>
  );
};
