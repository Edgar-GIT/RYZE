import {
  ArrowRight,
  Dumbbell,
  Eye,
  Library,
  Mail,
  Menu,
  MessageSquareText,
  Newspaper,
  UserRound,
  X
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useEffect, useRef, useState, type CSSProperties } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import { LanguageSwitcher } from "@/components/language_switcher/language_switcher";
import { PUBLIC_NAVIGATION_ITEMS } from "@/constants/navigation";
import { useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";
import { apiGet } from "@utils/http_client";

import styles from "./navbar.module.css";

const SCROLL_RANGE = 140;
const SCROLL_IDLE_MS = 220;

const NAV_ICONS: Record<string, LucideIcon> = {
  "/services": Dumbbell,
  "/exercises": Library,
  "/contact": Mail,
  "/feedback": MessageSquareText,
  "/our-vision": Eye,
  "/feed": Newspaper
};

export const Navbar = () => {
  const { t } = useI18n();
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [scrollProgress, setScrollProgress] = useState(0);
  const [isScrolling, setIsScrolling] = useState(false);
  const location = useLocation();
  const scrollIdleTimeout = useRef<number | null>(null);
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  // Signed-in visitors get a way back to their area instead of "Log in" and
  // "Start free". The answer comes from the session cookie (/me); a visitor
  // without a session simply keeps the sign-in buttons.
  const [signedIn, setSignedIn] = useState(false);

  useEffect(() => {
    let cancelled = false;
    apiGet("/me")
      .then(() => {
        if (!cancelled) setSignedIn(true);
      })
      .catch(() => {
        if (!cancelled) setSignedIn(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    setIsMenuOpen(false);
  }, [location.pathname]);

  useEffect(() => {
    if (!isMenuOpen) {
      return;
    }

    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsMenuOpen(false);
        menuButtonRef.current?.focus();
      }
    };

    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [isMenuOpen]);

  useEffect(() => {
    const clearIdleTimer = () => {
      if (scrollIdleTimeout.current !== null) {
        window.clearTimeout(scrollIdleTimeout.current);
        scrollIdleTimeout.current = null;
      }
    };

    const updateScrollState = () => {
      setScrollProgress(Math.min(1, window.scrollY / SCROLL_RANGE));
      setIsScrolling(true);
      clearIdleTimer();
      scrollIdleTimeout.current = window.setTimeout(() => {
        setIsScrolling(false);
        scrollIdleTimeout.current = null;
      }, SCROLL_IDLE_MS);
    };

    updateScrollState();
    setIsScrolling(false);
    clearIdleTimer();

    window.addEventListener("scroll", updateScrollState, { passive: true });

    return () => {
      clearIdleTimer();
      window.removeEventListener("scroll", updateScrollState);
    };
  }, []);

  const toggleMenu = () => {
    setIsMenuOpen((currentValue) => !currentValue);
  };

  return (
    <header
      className={joinClassNames(
        styles.header,
        scrollProgress > 0.08 && styles.headerScrolled,
        isScrolling && styles.headerCompact
      )}
      style={{ "--nav-scroll": String(scrollProgress) } as CSSProperties}
    >
      <nav className={styles.navbar} aria-label={t("publicNav.primary")}>
        <Link className={styles.brand} to="/" aria-label={t("publicNav.home")}>
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </Link>

        <div className={styles.desktopLinks}>
          {PUBLIC_NAVIGATION_ITEMS.map((item) => {
            const Icon = NAV_ICONS[item.to];
            const label = t(item.labelKey);

            return (
              <NavLink
                key={item.to}
                to={item.to}
                className={styles.navLink}
                activeClassName={styles.activeLink}
                exact={item.to === "/"}
                aria-label={label}
                title={label}
              >
                {Icon ? <Icon aria-hidden="true" className={styles.navIcon} strokeWidth={1.7} /> : null}
                <span className={styles.navLabel}>{t(item.shortKey ?? item.labelKey)}</span>
              </NavLink>
            );
          })}
        </div>

        <LanguageSwitcher variant="compact" className={styles.language} />

        <div className={styles.authActions}>
          {signedIn ? (
            <Link className={styles.startButton} to="/profile">
              <UserRound aria-hidden="true" strokeWidth={1.9} />
              <span>{t("publicNav.myAccount")}</span>
            </Link>
          ) : (
            <>
              <Link className={styles.loginButton} to="/login">
                <UserRound aria-hidden="true" strokeWidth={1.7} />
                <span>{t("publicNav.login")}</span>
              </Link>
              <Link className={styles.startButton} to="/register">
                <span>{t("publicNav.start")}</span>
                <ArrowRight aria-hidden="true" strokeWidth={2} />
              </Link>
            </>
          )}
        </div>

        <button
          ref={menuButtonRef}
          className={styles.menuButton}
          type="button"
          aria-label={isMenuOpen ? t("publicNav.closeMenu") : t("publicNav.openMenu")}
          aria-controls="mobile-navigation"
          aria-expanded={isMenuOpen}
          onClick={toggleMenu}
        >
          {isMenuOpen ? <X aria-hidden="true" /> : <Menu aria-hidden="true" />}
        </button>
      </nav>

      <div
        id="mobile-navigation"
        className={joinClassNames(styles.mobilePanel, isMenuOpen && styles.mobilePanelOpen)}
      >
        {PUBLIC_NAVIGATION_ITEMS.map((item) => {
          const Icon = NAV_ICONS[item.to];

          return (
            <NavLink
              key={item.to}
              to={item.to}
              className={styles.mobileLink}
              activeClassName={styles.activeMobileLink}
              exact={item.to === "/"}
            >
              {Icon ? <Icon aria-hidden="true" strokeWidth={1.7} /> : null}
              <span>{t(item.labelKey)}</span>
            </NavLink>
          );
        })}
        <div className={joinClassNames(styles.mobileAuth, signedIn && styles.mobileAuthSingle)}>
          {signedIn ? (
            <Link className={styles.startButton} to="/profile">
              <UserRound aria-hidden="true" strokeWidth={1.9} />
              <span>{t("publicNav.myAccount")}</span>
            </Link>
          ) : (
            <>
              <Link className={styles.loginButton} to="/login">
                <UserRound aria-hidden="true" strokeWidth={1.7} />
                <span>{t("publicNav.login")}</span>
              </Link>
              <Link className={styles.startButton} to="/register">
                <span>{t("publicNav.start")}</span>
                <ArrowRight aria-hidden="true" strokeWidth={2} />
              </Link>
            </>
          )}
        </div>
      </div>
    </header>
  );
};
