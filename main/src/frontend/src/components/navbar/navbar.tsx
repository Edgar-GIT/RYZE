import { Menu, UserRound, X } from "lucide-react";
import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties
} from "react";
import { Link, NavLink, useLocation } from "react-router-dom";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import { PUBLIC_NAVIGATION_ITEMS } from "@/constants/navigation";
import { joinClassNames } from "@utils/class_names";

import styles from "./navbar.module.css";

const SCROLL_RANGE = 140;

export const Navbar = () => {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [scrollProgress, setScrollProgress] = useState(0);
  const [indicator, setIndicator] = useState({ left: 0, width: 0, visible: false });
  const location = useLocation();
  const linksRef = useRef<HTMLDivElement>(null);
  const itemRefs = useRef<Record<string, HTMLAnchorElement | null>>({});

  useEffect(() => {
    setIsMenuOpen(false);
  }, [location.pathname]);

  useEffect(() => {
    const updateScrollState = () => {
      setScrollProgress(Math.min(1, window.scrollY / SCROLL_RANGE));
    };

    updateScrollState();
    window.addEventListener("scroll", updateScrollState, { passive: true });

    return () => {
      window.removeEventListener("scroll", updateScrollState);
    };
  }, []);

  useLayoutEffect(() => {
    const updateIndicator = () => {
      const activeItem = itemRefs.current[location.pathname];
      const linksElement = linksRef.current;

      if (!activeItem || !linksElement) {
        setIndicator((current) => ({ ...current, visible: false }));
        return;
      }

      const linksRect = linksElement.getBoundingClientRect();
      const itemRect = activeItem.getBoundingClientRect();

      setIndicator({
        left: itemRect.left - linksRect.left,
        width: itemRect.width,
        visible: true
      });
    };

    updateIndicator();
    window.addEventListener("resize", updateIndicator);

    return () => {
      window.removeEventListener("resize", updateIndicator);
    };
  }, [location.pathname, isMenuOpen]);

  const toggleMenu = () => {
    setIsMenuOpen((currentValue) => !currentValue);
  };

  return (
    <header
      className={joinClassNames(styles.header, scrollProgress > 0.08 && styles.headerScrolled)}
      style={{ "--nav-scroll": String(scrollProgress) } as CSSProperties}
    >
      <nav className={styles.navbar} aria-label="Primary navigation">
        <Link className={styles.brand} to="/" aria-label="RYZE home">
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </Link>

        <div className={styles.desktopLinks} ref={linksRef}>
          {PUBLIC_NAVIGATION_ITEMS.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={styles.navLink}
              activeClassName={styles.activeLink}
              exact={item.to === "/"}
              innerRef={(element) => {
                itemRefs.current[item.to] = element;
              }}
            >
              {item.label}
            </NavLink>
          ))}
          <span
            className={joinClassNames(
              styles.activeIndicator,
              indicator.visible && styles.activeIndicatorVisible
            )}
            style={{
              transform: `translateX(${indicator.left}px)`,
              width: indicator.width
            }}
            aria-hidden="true"
          />
        </div>

        <NavLink
          className={styles.profileLink}
          activeClassName={styles.activeProfile}
          to="/profile"
          aria-label="Profile"
          title="Profile"
        >
          <UserRound aria-hidden="true" />
        </NavLink>

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
        {PUBLIC_NAVIGATION_ITEMS.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={styles.mobileLink}
            activeClassName={styles.activeMobileLink}
            exact={item.to === "/"}
          >
            {item.label}
          </NavLink>
        ))}
        <NavLink
          to="/profile"
          className={joinClassNames(styles.mobileLink, styles.mobileProfile)}
          activeClassName={styles.activeMobileLink}
        >
          <UserRound aria-hidden="true" />
          <span>Profile</span>
        </NavLink>
      </div>
    </header>
  );
};
