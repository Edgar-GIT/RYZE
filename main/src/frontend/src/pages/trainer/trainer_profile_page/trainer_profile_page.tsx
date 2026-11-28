import { useCallback, useEffect, useState } from "react";
import { RefreshCw } from "lucide-react";

import { AppLoading } from "@/components/app_shell/app_loading";
import type { MemberUser } from "@/components/app_shell/member_shell";
import { Button } from "@/components/button/button";
import { useI18n } from "@/i18n/i18n";
import { fetchTrainerProfile } from "@/services/trainer_api";
import { joinClassNames } from "@utils/class_names";
import { apiGet } from "@utils/http_client";

import { ProfileDetailsCard } from "@/pages/account_settings_page/profile_details_card";
import { PublicProfileCard } from "@/pages/account_settings_page/public_profile_card";

import styles from "@/components/trainer/trainer_page.module.css";
import pageStyles from "./trainer_profile_page.module.css";

type State =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; me: MemberUser };

const COPY = {
  en: {
    eyebrow: "Trainer",
    title: "Profile",
    lead: "Your RYZE identity, edited right here: the same photo, name, bio and @username everywhere, as a member and as a coach.",
    loadError: "Unable to load your profile right now.",
    retry: "Try again"
  },
  pt: {
    eyebrow: "Treinador",
    title: "Perfil",
    lead: "A tua identidade na RYZE, editada aqui mesmo: a mesma foto, nome, bio e @referência em todo o lado, como membro e como treinador.",
    loadError: "Não foi possível carregar o teu perfil agora.",
    retry: "Tentar de novo"
  }
} as const;

// TrainerProfilePage edits the trainer's one RYZE identity in place (owner
// requests, 2026-11-21 and 2026-11-28): the same profile cards as the account
// settings, centred. The views and "public coach profile" blocks were removed
// (owner request, 2026-11-28); the sidebar links to the public profile.
export const TrainerProfilePage = () => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const [state, setState] = useState<State>({ status: "loading" });

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      // The trainer call confirms the trainer role before showing the page.
      const [, me] = await Promise.all([fetchTrainerProfile(), apiGet<MemberUser>("/me")]);
      setState({ status: "ready", me });
    } catch {
      setState({ status: "error" });
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (state.status === "loading") {
    return <AppLoading />;
  }

  if (state.status === "error") {
    return (
      <section className={styles.stateCard}>
        <h1 className={styles.stateTitle}>{text.title}</h1>
        <p className={styles.stateText}>{text.loadError}</p>
        <div className={styles.actions}>
          <Button variant="secondary" size="small" icon={<RefreshCw size={15} />} onClick={() => void load()}>
            {text.retry}
          </Button>
        </div>
      </section>
    );
  }

  const { me } = state;

  return (
    <div className={pageStyles.page}>
      <header className={joinClassNames(styles.pageHeading, pageStyles.heading)}>
        <div>
          <p className={styles.eyebrow}>{text.eyebrow}</p>
          <h1 className={styles.title}>{text.title}</h1>
          <p className={styles.subtitle}>{text.lead}</p>
        </div>
      </header>

      {/* Edited right here, like the account settings (owner request,
          2026-11-28): photo, nickname, bio and social links. */}
      <div className={pageStyles.editCards}>
        <PublicProfileCard user={me} onChanged={() => void load()} />
        <ProfileDetailsCard />
      </div>
    </div>
  );
};
