import { LogOut, type LucideIcon } from "lucide-react";
import { useEffect, useId, useRef, useState, type CSSProperties, type KeyboardEvent } from "react";
import { Link, useLocation } from "react-router-dom";

import { joinClassNames } from "@utils/class_names";

import styles from "./user_menu.module.css";

export interface ShellUser {
  name: string;
  email?: string;
  // A short role line under the name (e.g. "Coach", "Management").
  role?: string;
}

export interface UserMenuItem {
  to: string;
  label: string;
  icon: LucideIcon;
}

interface UserMenuProps {
  user: ShellUser | null;
  items: UserMenuItem[];
  // Accessible name of the trigger, e.g. "Account menu".
  label: string;
  signOutLabel: string;
  signingOutLabel: string;
  onSignOut: () => Promise<void> | void;
  // "avatar": the round top-bar button; "card": the sidebar account card.
  variant: "avatar" | "card";
  className?: string;
}

export const initialsOf = (name: string) =>
  name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("") || "R";

// UserMenu is the account button of the signed-in areas. A click opens a
// small menu with the account shortcuts and "Sign out". It follows the menu
// button pattern: arrow keys move between items, Escape closes and returns
// focus to the button, a click outside closes it.
export const UserMenu = ({
  user,
  items,
  label,
  signOutLabel,
  signingOutLabel,
  onSignOut,
  variant,
  className
}: UserMenuProps) => {
  const [open, setOpen] = useState(false);
  const [position, setPosition] = useState<CSSProperties>({});
  const [signingOut, setSigningOut] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const menuId = useId();
  const { pathname } = useLocation();

  useEffect(() => {
    setOpen(false);
  }, [pathname]);

  // The menu is fixed to the viewport (measured from the button) so the
  // scrolling sidebar never clips it.
  useEffect(() => {
    if (!open) {
      return;
    }
    const place = () => {
      const rect = buttonRef.current?.getBoundingClientRect();
      if (!rect) {
        return;
      }
      setPosition(
        variant === "card"
          ? { left: Math.max(12, rect.left), bottom: window.innerHeight - rect.top + 10 }
          : { right: Math.max(12, window.innerWidth - rect.right), top: rect.bottom + 10 }
      );
    };
    place();
    window.addEventListener("resize", place);
    return () => window.removeEventListener("resize", place);
  }, [open, variant]);

  useEffect(() => {
    if (!open) {
      return;
    }
    const first = menuRef.current?.querySelector<HTMLElement>("[role='menuitem']");
    first?.focus();

    const onPointerDown = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const onKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
        buttonRef.current?.focus();
      }
    };
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  const moveFocus = (event: KeyboardEvent<HTMLDivElement>) => {
    const entries = Array.from(menuRef.current?.querySelectorAll<HTMLElement>("[role='menuitem']") ?? []);
    if (entries.length === 0) {
      return;
    }
    const index = entries.indexOf(document.activeElement as HTMLElement);
    let next = -1;
    if (event.key === "ArrowDown") {
      next = (index + 1) % entries.length;
    } else if (event.key === "ArrowUp") {
      next = (index - 1 + entries.length) % entries.length;
    } else if (event.key === "Home") {
      next = 0;
    } else if (event.key === "End") {
      next = entries.length - 1;
    } else if (event.key === "Tab") {
      setOpen(false);
      return;
    }
    if (next >= 0) {
      event.preventDefault();
      entries[next]?.focus();
    }
  };

  const handleSignOut = async () => {
    if (signingOut) {
      return;
    }
    setSigningOut(true);
    try {
      await onSignOut();
    } finally {
      setSigningOut(false);
    }
  };

  const initials = initialsOf(user?.name ?? "");

  return (
    <div
      ref={rootRef}
      className={joinClassNames(styles.root, variant === "card" ? styles.rootCard : styles.rootAvatar, className)}
    >
      <button
        ref={buttonRef}
        type="button"
        className={variant === "card" ? styles.card : styles.avatarButton}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        aria-label={label}
        title={variant === "avatar" ? label : undefined}
        onClick={() => setOpen((value) => !value)}
        onKeyDown={(event) => {
          if (event.key === "ArrowDown" && !open) {
            event.preventDefault();
            setOpen(true);
          }
        }}
      >
        <span className={styles.avatar} aria-hidden="true">
          {initials}
        </span>
        {variant === "card" ? (
          <span className={styles.cardText}>
            <strong>{user?.name ?? "…"}</strong>
            {user?.email ? <span>{user.email}</span> : null}
          </span>
        ) : null}
      </button>

      {open ? (
        <div
          ref={menuRef}
          id={menuId}
          className={joinClassNames(styles.menu, variant === "card" ? styles.menuUp : styles.menuDown)}
          style={position}
          role="menu"
          aria-label={label}
          onKeyDown={moveFocus}
        >
          <div className={styles.head}>
            <span className={styles.headAvatar} aria-hidden="true">
              {initials}
            </span>
            <span className={styles.headText}>
              <strong>{user?.name ?? "…"}</strong>
              {user?.email ? <span>{user.email}</span> : null}
              {user?.role ? <em>{user.role}</em> : null}
            </span>
          </div>
          <div className={styles.items}>
            {items.map((item) => {
              const Icon = item.icon;
              return (
                <Link
                  key={item.to}
                  to={item.to}
                  className={styles.item}
                  role="menuitem"
                  tabIndex={-1}
                  onClick={() => setOpen(false)}
                >
                  <Icon aria-hidden="true" strokeWidth={1.9} />
                  <span>{item.label}</span>
                </Link>
              );
            })}
          </div>
          <button
            type="button"
            className={joinClassNames(styles.item, styles.signOut)}
            role="menuitem"
            tabIndex={-1}
            disabled={signingOut}
            onClick={() => void handleSignOut()}
          >
            <LogOut aria-hidden="true" strokeWidth={1.9} />
            <span>{signingOut ? signingOutLabel : signOutLabel}</span>
          </button>
        </div>
      ) : null}
    </div>
  );
};
