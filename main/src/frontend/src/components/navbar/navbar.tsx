import {
  ArrowRight,
  Dumbbell,
  Eye,
  Mail,
  Menu,
  MessageSquareText,
  UserRound,
  X
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useEffect, useRef, useState, type CSSProperties } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import { PUBLIC_NAVIGATION_ITEMS } from "@/constants/navigation";
import { joinClassNames } from "@utils/class_names";

import styles from "./navbar.module.css";

const SCROLL_RANGE = 140;
const SCROLL_IDLE_MS = 220;

const NAV_ICONS: Record<string, LucideIcon> = {
  "/services": Dumbbell,
  "/contact": Mail,
  "/feedback": MessageSquareText,
  "/our-vision": Eye
};

export const Navbar = () => {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [scrollProgress, setScrollProgress] = useState(0);
  const [isScrolling, setIsScrolling] = useState(false);
  const location = useLocation();
  const scrollIdleTimeout = useRef<number | null>(null);

  useEffect(() => {
    setIsMenuOpen(false);
  }, [location.pathname]);

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
      <nav className={styles.navbar} aria-label="Primary navigation">
        <Link className={styles.brand} to="/" aria-label="RYZE home">
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </Link>

        <div className={styles.desktopLinks}>
          {PUBLIC_NAVIGATION_ITEMS.map((item) => {
            const Icon = NAV_ICONS[item.to];

            return (
              <NavLink
                key={item.to}
                to={item.to}
                className={styles.navLink}
                activeClassName={styles.activeLink}
                exact={item.to === "/"}
                aria-label={item.label}
                title={item.label}
              >
                {Icon ? <Icon aria-hidden="true" className={styles.navIcon} strokeWidth={1.7} /> : null}
                <span className={styles.navLabel}>{item.label}</span>
              </NavLink>
            );
          })}
        </div>

        <div className={styles.authActions}>
          <Link className={styles.loginButton} to="/login">
            <UserRound aria-hidden="true" strokeWidth={1.7} />
            <span>Log in</span>
          </Link>
          <Link className={styles.startButton} to="/register">
            <span>Start free</span>
            <ArrowRight aria-hidden="true" strokeWidth={2} />
          </Link>
        </div>

        <button
          className={styles.menuButton}
          type="button"
          aria-label={isMenuOpen ? "Close navigation" : "Open navigation"}
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
              <span>{item.label}</span>
            </NavLink>
          );
        })}
        <div className={styles.mobileAuth}>
          <Link className={styles.loginButton} to="/login">
            <UserRound aria-hidden="true" strokeWidth={1.7} />
            <span>Log in</span>
          </Link>
          <Link className={styles.startButton} to="/register">
            <span>Start free</span>
            <ArrowRight aria-hidden="true" strokeWidth={2} />
          </Link>
        </div>
      </div>
    </header>
  );
};
