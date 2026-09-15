import { useEffect, useState } from "react";
import { Star } from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2EmptyState } from "@/components/admin2/admin2_empty_state/admin2_empty_state";
import { fetchAdminFeedback } from "@/services/admin2_api";

import styles from "./admin2_feedback_page.module.css";

export default function Admin2FeedbackPage() {
  const [available, setAvailable] = useState<boolean | null>(null);
  const [reason, setReason] = useState("");

  useEffect(() => {
    let cancelled = false;

    fetchAdminFeedback().then((result) => {
      if (!cancelled) {
        setAvailable(result.available);
        setReason(result.reason);
      }
    });

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <>
      <Admin2PageHeader
        eyebrow="Engagement"
        title="Feedback"
        description="How members feel about the platform and the programs they run."
      />

      <Admin2Section title="Member feedback" subtitle="Ratings and messages collected from members">
        {available === null ? (
          <div className={styles.pageState}>
            <p className={styles.muted}>Loading feedback…</p>
          </div>
        ) : (
          <Admin2EmptyState
            icon={Star}
            title="Feedback is not part of the community scope yet."
            message={reason}
            action={<span className={styles.muted}>Reviews will surface with programme ratings in a later milestone.</span>}
          />
        )}
      </Admin2Section>
    </>
  );
}