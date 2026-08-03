import { Menu, UserRound, X } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import { PUBLIC_NAVIGATION_ITEMS } from "@/constants/navigation";
import { joinClassNames } from "@utils/class_names";

import styles from "./navbar.module.css";

export const Navbar = () => {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isScrolled, setIsScrolled] = useState(false);
  const location = useLocation();

  useEffect(() => {
    setIsMenuOpen(false);
  }, [location.pathname]);

  useEffect(() => {
    const updateScrollState = () => {
      setIsScrolled(window.scrollY > 18);
    };

    updateScrollState();
    window.addEventListener("scroll", updateScrollState, { passive: true });

    return () => {
      window.removeEventListener("scroll", updateScrollState);
    };
  }, []);

  const toggleMenu = () => {
    setIsMenuOpen((currentValue) => !currentValue);
  };

  return (
    <header className={joinClassNames(styles.header, isScrolled && styles.headerScrolled)}>
      <nav className={styles.navbar} aria-label="Primary navigation">
        <Link className={styles.brand} to="/" aria-label="RYZE home">
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </Link>

        <div className={styles.desktopLinks}>
          {PUBLIC_NAVIGATION_ITEMS.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={styles.navLink}
              activeClassName={styles.activeLink}
            >
              {item.label}
            </NavLink>
          ))}
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
        className={joinClassNames(
          styles.mobilePanel,
          isScrolled && styles.mobilePanelScrolled,
          isMenuOpen && styles.mobilePanelOpen
        )}
      >
        {PUBLIC_NAVIGATION_ITEMS.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={styles.mobileLink}
            activeClassName={styles.activeMobileLink}
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
