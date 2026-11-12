import { Link } from "react-router-dom";
import { UserRound } from "lucide-react";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import { useI18n } from "@/i18n/i18n";
import { NotificationBell } from "@/components/notification_bell/notification_bell";

import styles from "./account_nav.module.css";

// AccountNav is the single authenticated account navigation bar used by the
// chrome-free account area (Profile, My Programs, Purchase History and Program
// Access). It intentionally renders no global chrome: the account pages stay
// full-screen and content-focused, matching the existing design language.
interface AccountNavProps {
  userName?: string;
}

export const AccountNav = ({ userName }: AccountNavProps) => {
  const { t } = useI18n();
  return (
    <header className={styles.navbar}>
      <Link className={styles.brand} to="/" aria-label="RYZE home">
        <BrandMark size="navigation" />
        <span>RYZE</span>
      </Link>

      <nav className={styles.centerNav} aria-label={t("account.nav.label")}>
        <Link className={styles.navLink} to="/profile">
          {t("account.nav.profile")}
        </Link>
        <Link className={styles.navLink} to="/services/my-programs">
          {t("account.nav.myPrograms")}
        </Link>
        <Link className={styles.navLink} to="/services/progress">
          {t("account.nav.progress")}
        </Link>
        <Link className={styles.navLink} to="/account/purchases">
          {t("account.nav.purchases")}
        </Link>
        <Link className={styles.navLink} to="/support">
          {t("account.nav.support")}
        </Link>
      </nav>

      <div className={styles.userArea}>
        {userName ? (
          <>
            <span className={styles.userName}>{userName}</span>
            <span className={styles.avatar} aria-hidden="true">
              <UserRound strokeWidth={1.6} />
            </span>
          </>
        ) : null}
        <NotificationBell />
      </div>
    </header>
  );
};