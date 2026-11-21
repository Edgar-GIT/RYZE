import { useCallback, useEffect, useState } from "react";
import { Eye, Pencil, RefreshCw, UserRound } from "lucide-react";

import { fetchCoachProfile } from "@/services/coaches_api";

import { AppLoading } from "@/components/app_shell/app_loading";
import type { MemberUser } from "@/components/app_shell/member_shell";
import { Button } from "@/components/button/button";
import { SOCIAL_NAMES, SocialIcon } from "@/components/social_icons/social_icons";
import { useI18n } from "@/i18n/i18n";
import { fetchProfileDetails, hasNickname, publicNameOf, type ProfileDetails } from "@/services/profile_api";
import { profilePath } from "@/services/public_profiles_api";
import { fetchTrainerProfile, type TrainerProfile } from "@/services/trainer_api";
import { joinClassNames } from "@utils/class_names";
import { apiGet, publicFileUrl } from "@utils/http_client";

import styles from "@/components/trainer/trainer_page.module.css";
import pageStyles from "./trainer_profile_page.module.css";

type State =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; trainer: TrainerProfile; me: MemberUser; details: ProfileDetails | null; views: number | null };

const COPY = {
  en: {
    eyebrow: "Trainer",
    title: "Profile",
    lead: "Your RYZE identity: the same photo, name and @username everywhere, as a member and as a coach.",
    loadError: "Unable to load your profile right now.",
    retry: "Try again",
    noBio: "You have not written a biography yet.",
    since: "Member since",
    status: "Status",
    active: "Active trainer",
    edit: "Edit profile",
    view: "View public profile",
    coachTitle: "Public coach profile",
    coachText: "Clients see your name and every plan you publish. It goes live with your first published plan; Test Mode accounts are never public.",
    coachOpen: "Open coach profile",
    viewsTitle: "Views",
    viewsText: "Every visit to your published plans. Only coaches have this number: the more views, the more popular you are.",
    viewsNone: "Publish your first plan to start counting views."
  },
  pt: {
    eyebrow: "Treinador",
    title: "Perfil",
    lead: "A tua identidade na RYZE: a mesma foto, nome e @referência em todo o lado, como membro e como treinador.",
    loadError: "Não foi possível carregar o teu perfil agora.",
    retry: "Tentar de novo",
    noBio: "Ainda não escreveste a tua biografia.",
    since: "Membro desde",
    status: "Estado",
    active: "Treinador ativo",
    edit: "Editar perfil",
    view: "Ver perfil público",
    coachTitle: "Perfil público de treinador",
    coachText: "Os clientes veem o teu nome e todos os planos que publicas. Fica ativo com o teu primeiro plano publicado; as contas do Test Mode nunca são públicas.",
    coachOpen: "Abrir perfil de treinador",
    viewsTitle: "Visualizações",
    viewsText: "Todas as visitas aos teus planos publicados. Só os treinadores têm este número: quantas mais views, mais popular és.",
    viewsNone: "Publica o teu primeiro plano para começares a contar visualizações."
  }
} as const;

// TrainerProfilePage shows the trainer's one RYZE identity (owner report,
// 2026-11-21: the page showed the registered name and the email and said the
// profile could not be edited). Editing happens in Account, shared with the
// member area.
export const TrainerProfilePage = () => {
  const { locale, formatDate, formatNumber } = useI18n();
  const text = COPY[locale];
  const [state, setState] = useState<State>({ status: "loading" });

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const [trainer, me, details] = await Promise.all([
        fetchTrainerProfile(),
        apiGet<MemberUser>("/me"),
        fetchProfileDetails().catch(() => null)
      ]);
      const views = await fetchCoachProfile(trainer.id)
        .then((coach) => coach.coach.total_views)
        .catch(() => null);
      setState({ status: "ready", trainer, me, details, views });
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

  const { trainer, me, details, views } = state;
  const name = publicNameOf(me);

  return (
    <div className={pageStyles.page}>
      <header className={joinClassNames(styles.pageHeading, pageStyles.heading)}>
        <div>
          <p className={styles.eyebrow}>{text.eyebrow}</p>
          <h1 className={styles.title}>{text.title}</h1>
          <p className={styles.subtitle}>{text.lead}</p>
        </div>
      </header>

      <section className={joinClassNames(styles.card, pageStyles.identityCard)}>
        <div className={pageStyles.identity}>
          {me.avatar_url ? (
            <img className={pageStyles.avatar} src={publicFileUrl(me.avatar_url)} alt="" />
          ) : (
            <span className={pageStyles.avatar} aria-hidden="true">
              <UserRound strokeWidth={1.6} />
            </span>
          )}
          <div className={styles.rowMain}>
            <span className={pageStyles.name}>{name}</span>
            {hasNickname(me) ? <span className={pageStyles.handle}>@{me.username}</span> : null}
          </div>
        </div>

        <p className={details?.bio ? pageStyles.bio : styles.hint}>{details?.bio || text.noBio}</p>

        {details && details.socials.length > 0 ? (
          <ul className={pageStyles.socials}>
            {details.socials.map((link) => (
              <li key={link.platform}>
                <a href={link.url} target="_blank" rel="noopener noreferrer nofollow" aria-label={`${SOCIAL_NAMES[link.platform]}: @${link.handle}`}>
                  <SocialIcon platform={link.platform} />
                </a>
              </li>
            ))}
          </ul>
        ) : null}

        <div className={pageStyles.metaGrid}>
          <div className={styles.rowMain}>
            <span className={styles.statLabel}>{text.since}</span>
            <span className={styles.rowTitle}>{formatDate(me.created_at, { year: "numeric", month: "long", day: "numeric" })}</span>
          </div>
          <div className={styles.rowMain}>
            <span className={styles.statLabel}>{text.status}</span>
            <span className={styles.rowTitle}>{text.active}</span>
          </div>
        </div>

        <div className={styles.actions}>
          <Button to="/account/settings" variant="primary" size="small" icon={<Pencil size={15} />} iconPosition="left">
            {text.edit}
          </Button>
          <Button to={profilePath(me.username)} variant="secondary" size="small" icon={<Eye size={15} />} iconPosition="left">
            {text.view}
          </Button>
        </div>
      </section>

      <section className={joinClassNames(styles.card, pageStyles.viewsCard)} aria-labelledby="trainer-views">
        <span id="trainer-views" className={styles.statLabel}>
          {text.viewsTitle}
        </span>
        <p className={pageStyles.views}>
          <Eye aria-hidden="true" />
          {formatNumber(views ?? 0)}
        </p>
        <p className={styles.hint}>{views === null ? text.viewsNone : text.viewsText}</p>
      </section>

      <section className={joinClassNames(styles.card, pageStyles.coachCard)}>
        <div className={styles.rowMain}>
          <span className={styles.statLabel}>{text.coachTitle}</span>
        </div>
        <p className={styles.hint}>{text.coachText}</p>
        <div className={styles.actions}>
          <Button to={`/coaches/${encodeURIComponent(trainer.id)}`} variant="secondary" size="small" icon={<Eye size={15} />} iconPosition="left">
            {text.coachOpen}
          </Button>
        </div>
      </section>
    </div>
  );
};
