import { coachInitials } from "@/services/coaches_api";
import { publicFileUrl } from "@utils/http_client";

import styles from "./coach_avatar.module.css";

// CoachAvatar is the coach's profile photo in the "Created by" circle, or
// their initials when they have no photo (owner request, 2026-11-27). The
// caller's class sets the size and the initials' colours.
export const CoachAvatar = ({ name, avatarUrl, className }: { name: string; avatarUrl?: string | null; className?: string }) => (
  <span className={className} aria-hidden="true">
    {avatarUrl ? <img className={styles.photo} src={publicFileUrl(avatarUrl)} alt="" loading="lazy" /> : coachInitials(name)}
  </span>
);
