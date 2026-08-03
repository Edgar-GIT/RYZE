import { Menu, UserRound, X } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";

import { BRAND_ASSETS } from "@/constants/brand_assets";
import { PUBLIC_NAVIGATION_ITEMS } from "@/constants/navigation";
import { joinClassNames } from "@utils/class_names";

import styles from "./navbar.module.css";

export const Navbar = () => {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const location = useLocation();

  useEffect(() => {
    setIsMenuOpen(false);
  }, [location.pathname]);

  const toggleMenu = () => {
    setIsMenuOpen((currentValue) => !currentValue);
  };

  return (
    <header className={styles.header}>
      <nav className={styles.navbar} aria-label="Primary navigation">
        <Link className={styles.brand} to="/" aria-label="RYZE home">
          <img src={BRAND_ASSETS.icon} alt="" aria-hidden="true" />
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
        className={joinClassNames(styles.mobilePanel, isMenuOpen && styles.mobilePanelOpen)}
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
