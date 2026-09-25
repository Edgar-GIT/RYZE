import { Link } from "react-router-dom";
import { UserRound } from "lucide-react";

import { BrandMark } from "@/components/brand_mark/brand_mark";

import styles from "./account_nav.module.css";

// AccountNav is the single authenticated account navigation bar used by the
// chrome-free account area (Profile, My Programs, Purchase History and Program
// Access). It intentionally renders no global chrome: the account pages stay
// full-screen and content-focused, matching the existing design language.
interface AccountNavProps {
  userName?: string;
}

export const AccountNav = ({ userName }: AccountNavProps) => {
  return (
    <header className={styles.navbar}>
      <Link className={styles.brand} to="/" aria-label="RYZE home">
        <BrandMark size="navigation" />
        <span>RYZE</span>
      </Link>

      <nav className={styles.centerNav} aria-label="Account">
        <Link className={styles.navLink} to="/profile">
          Profile
        </Link>
        <Link className={styles.navLink} to="/services/my-programs">
          My Programs
        </Link>
        <Link className={styles.navLink} to="/account/purchases">
          Purchase History
        </Link>
      </nav>

      {userName ? (
        <div className={styles.userArea}>
          <span className={styles.userName}>{userName}</span>
          <span className={styles.avatar} aria-hidden="true">
            <UserRound strokeWidth={1.6} />
          </span>
        </div>
      ) : (
        <div className={styles.spacer} aria-hidden="true" />
      )}
    </header>
  );
};