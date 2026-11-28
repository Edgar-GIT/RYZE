import { ArrowLeft, BadgeCheck, ClipboardList, CalendarDays, Dumbbell, Eye, Medal, Pencil, RefreshCw, UserCheck, UserPlus, Users } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { LEVEL_NAMES, LevelBadge } from "@/components/lifter/lifter";
import { SOCIAL_NAMES, SocialIcon } from "@/components/social_icons/social_icons";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { fetchCoachProfile, type CoachProfile } from "@/services/coaches_api";
import ui from "@/components/public_ui/public_ui.module.css";
import { useBackTarget } from "@/hooks/use_back_target";
import { useI18n } from "@/i18n/i18n";
import { rememberAfterLogin } from "@/services/after_login";
import {
  fetchFollowers,
  fetchFollowing,
  fetchPublicProfile,
  followUser,
  profilePath,
  unfollowUser,
  type PublicPerson,
  type PublicProfile
} from "@/services/public_profiles_api";
import { joinClassNames } from "@utils/class_names";
import { ApiError, publicFileUrl } from "@utils/http_client";

import styles from "./public_profile_page.module.css";

const COPY = {
  en: {
    loading: "Loading profile…",
    notFound: "There is no RYZE profile with this @username.",
    loadError: "We could not load this profile.",
    retry: "Try again",
    since: (date: string) => `Member since ${date}`,
    workouts: "Workouts",
    followers: "Followers",
    following: "Following",
    medals: "Record medals",
    views: "Coach views",
    follow: "Follow",
    unfollow: "Following",
    unfollowHint: "Stop following",
    signInToFollow: "Sign in to follow",
    edit: "Edit my profile",
    coach: "Coach profile",
    coachBadge: "RYZE coach",
    coachText: "Certified RYZE coach: creates training plans for RYZE members.",
    plansCount: "Plans created",
    subscriptions: "Subscriptions",
    age: (years: number) => `${years} years old`,
    training: (years: number) => (years < 1 ? "Started training this year" : years === 1 ? "1 year of training" : `${years} years of training`),
    socials: "Social networks",
    empty: { followers: "Nobody follows this profile yet.", following: "This profile follows nobody yet." },
    followError: "We could not update following. Please try again."
  },
  pt: {
    loading: "A carregar o perfil…",
    notFound: "Não há nenhum perfil RYZE com esta @referência.",
    loadError: "Não foi possível carregar este perfil.",
    retry: "Tentar de novo",
    since: (date: string) => `Membro desde ${date}`,
    workouts: "Treinos",
    followers: "Seguidores",
    following: "A seguir",
    medals: "Medalhas de recorde",
    views: "Visualizações de treinador",
    follow: "Seguir",
    unfollow: "A seguir",
    unfollowHint: "Deixar de seguir",
    signInToFollow: "Inicia sessão para seguir",
    edit: "Editar o meu perfil",
    coach: "Perfil de treinador",
    coachBadge: "Treinador RYZE",
    coachText: "Treinador certificado RYZE: cria planos de treino para os membros da RYZE.",
    plansCount: "Planos criados",
    subscriptions: "Assinaturas",
    age: (years: number) => `${years} anos`,
    training: (years: number) => (years < 1 ? "Começou a treinar este ano" : years === 1 ? "1 ano de treino" : `${years} anos de treino`),
    socials: "Redes sociais",
    empty: { followers: "Ainda ninguém segue este perfil.", following: "Este perfil ainda não segue ninguém." },
    followError: "Não foi possível atualizar. Tenta de novo."
  }
} as const;

const Avatar = ({ person, large = false }: { person: PublicPerson; large?: boolean }) => {
  const name = person.nickname?.trim() || person.username;
  return person.avatar_url ? (
    <img className={joinClassNames(styles.avatar, large && styles.avatarLarge)} src={publicFileUrl(person.avatar_url)} alt="" />
  ) : (
    <span className={joinClassNames(styles.avatar, large && styles.avatarLarge)} aria-hidden="true">
      {name.charAt(0).toUpperCase()}
    </span>
  );
};

type ListKind = "followers" | "following";

// PublicProfilePage is a person's public profile (owner request, 2026-11-21),
// open to everyone: photo, nickname (else the @username, shown once), member
// since, level, workouts, record medals, followers and following.
export const PublicProfilePage = () => {
  const { username } = useParams<{ username: string }>();
  const { locale, formatDate, formatNumber } = useI18n();
  const text = COPY[locale];
  const back = useBackTarget();
  const [profile, setProfile] = useState<PublicProfile | null>(null);
  const [state, setState] = useState<"loading" | "ready" | "missing" | "error">("loading");
  const [busy, setBusy] = useState(false);
  const [followError, setFollowError] = useState("");
  const [list, setList] = useState<{ kind: ListKind; people: PublicPerson[] | null } | null>(null);
  // A coach's profile is the member profile plus the coach badge and the
  // plans they created (owner request, 2026-11-27).
  const [coach, setCoach] = useState<CoachProfile | null>(null);

  const load = useCallback(async () => {
    setState("loading");
    setList(null);
    setCoach(null);
    try {
      const loaded = await fetchPublicProfile(username);
      setProfile(loaded);
      setState("ready");
      if (loaded.coach_id) {
        setCoach(await fetchCoachProfile(loaded.coach_id).catch(() => null));
      }
    } catch (error) {
      setProfile(null);
      setState(error instanceof ApiError && error.status === 404 ? "missing" : "error");
    }
  }, [username]);

  useEffect(() => {
    void load();
  }, [load]);

  const openList = async (kind: ListKind) => {
    if (list?.kind === kind) {
      setList(null);
      return;
    }
    setList({ kind, people: null });
    try {
      const people = kind === "followers" ? await fetchFollowers(username) : await fetchFollowing(username);
      setList({ kind, people });
    } catch {
      setList({ kind, people: [] });
    }
  };

  const toggleFollow = async () => {
    if (!profile) return;
    setBusy(true);
    setFollowError("");
    const following = !profile.viewer.following;
    try {
      if (following) await followUser(profile.username);
      else await unfollowUser(profile.username);
      setProfile({
        ...profile,
        viewer: { ...profile.viewer, following },
        stats: { ...profile.stats, followers: Math.max(0, profile.stats.followers + (following ? 1 : -1)) }
      });
      if (list?.kind === "followers") setList(null);
    } catch {
      setFollowError(text.followError);
    } finally {
      setBusy(false);
    }
  };

  const name = profile ? profile.nickname?.trim() || profile.username : "";
  const about = profile?.about ?? null;
  const facts = about
    ? [
        about.age != null ? text.age(about.age) : "",
        about.training_years != null ? text.training(about.training_years) : "",
        about.weight_kg != null ? `${formatNumber(about.weight_kg, { maximumFractionDigits: 1 })} kg` : ""
      ].filter(Boolean)
    : [];
  const levelName = profile && profile.stats.level.key ? LEVEL_NAMES[locale][profile.stats.level.key] : "";

  return (
    <PageWrapper className={joinClassNames(ui.page, styles.page)}>
      <Container size="default" className={styles.container}>
        <div>
          <Button to={back.to} onClick={back.onClick} variant="ghost" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
            {back.label}
          </Button>
        </div>

        {state === "loading" ? <p className={styles.state}>{text.loading}</p> : null}
        {state === "missing" || state === "error" ? (
          <div className={styles.state}>
            <p>{state === "missing" ? text.notFound : text.loadError}</p>
            {state === "error" ? (
              <Button variant="secondary" size="small" icon={<RefreshCw size={15} />} onClick={() => void load()}>
                {text.retry}
              </Button>
            ) : null}
          </div>
        ) : null}

        {state === "ready" && profile ? (
          <>
            <header className={styles.hero}>
              <Avatar person={profile} large />
              <div className={styles.identity}>
                <h1 className={styles.name}>{name}</h1>
                {profile.nickname?.trim() ? <p className={styles.handle}>@{profile.username}</p> : null}
                <p className={styles.meta}>
                  <CalendarDays aria-hidden="true" />
                  {text.since(formatDate(profile.member_since, { month: "long", year: "numeric" }))}
                </p>
                <div className={styles.chips}>
                  {profile.stats.level.key ? (
                    <span className={styles.levelChip}>
                      <LevelBadge level={{ rank: profile.stats.level.rank, key: profile.stats.level.key }} size="sm" />
                      {levelName}
                    </span>
                  ) : null}
                  {profile.coach_id ? (
                    <span className={styles.coachChip}>
                      <BadgeCheck aria-hidden="true" /> {text.coachBadge}
                    </span>
                  ) : null}
                </div>
                {profile.coach_id ? <p className={styles.meta}>{text.coachText}</p> : null}
              </div>
              {about && (about.bio || facts.length > 0 || about.socials.length > 0) ? (
                <div className={styles.about}>
                  {about.bio ? <p className={styles.bio}>{about.bio}</p> : null}
                  {facts.length > 0 ? (
                    <ul className={styles.facts}>
                      {facts.map((fact) => (
                        <li key={fact}>{fact}</li>
                      ))}
                    </ul>
                  ) : null}
                  {about.socials.length > 0 ? (
                    <ul className={styles.socials} aria-label={text.socials}>
                      {about.socials.map((link) => (
                        <li key={link.platform}>
                          <a
                            className={styles.social}
                            href={link.url}
                            target="_blank"
                            rel="noopener noreferrer nofollow ugc"
                            aria-label={`${SOCIAL_NAMES[link.platform]}: @${link.handle}`}
                            title={`${SOCIAL_NAMES[link.platform]} · @${link.handle}`}
                          >
                            <SocialIcon platform={link.platform} />
                          </a>
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </div>
              ) : null}
              <div className={styles.actions}>
                {profile.viewer.is_self ? (
                  <Button to="/account/settings" variant="secondary" size="small" icon={<Pencil size={15} />} iconPosition="left">
                    {text.edit}
                  </Button>
                ) : profile.viewer.signed_in ? (
                  <button
                    type="button"
                    className={joinClassNames(styles.follow, profile.viewer.following && styles.following)}
                    aria-pressed={profile.viewer.following}
                    title={profile.viewer.following ? text.unfollowHint : undefined}
                    disabled={busy}
                    onClick={() => void toggleFollow()}
                  >
                    {profile.viewer.following ? <UserCheck aria-hidden="true" /> : <UserPlus aria-hidden="true" />}
                    {profile.viewer.following ? text.unfollow : text.follow}
                  </button>
                ) : (
                  <Link className={styles.follow} to="/login" onClick={() => rememberAfterLogin(profilePath(profile.username))}>
                    <UserPlus aria-hidden="true" /> {text.signInToFollow}
                  </Link>
                )}
              </div>
              {followError ? (
                <p className={styles.error} role="alert">
                  {followError}
                </p>
              ) : null}
            </header>

            <dl className={styles.stats}>
              <div className={styles.stat}>
                <dt>
                  <Dumbbell aria-hidden="true" /> {text.workouts}
                </dt>
                <dd>{formatNumber(profile.stats.workouts)}</dd>
              </div>
              {(["followers", "following"] as const).map((kind) => (
                <div key={kind} className={joinClassNames(styles.stat, list?.kind === kind && styles.statOpen)}>
                  <dt>
                    <Users aria-hidden="true" /> {text[kind]}
                  </dt>
                  <dd>
                    <button type="button" className={styles.statButton} aria-expanded={list?.kind === kind} onClick={() => void openList(kind)}>
                      {formatNumber(profile.stats[kind])}
                    </button>
                  </dd>
                </div>
              ))}
              <div className={styles.stat}>
                <dt>
                  <Medal aria-hidden="true" /> {text.medals}
                </dt>
                <dd>{formatNumber(profile.stats.medals)}</dd>
              </div>
              {/* A coach's counters (owner request, 2026-11-28): plans created
                  and subscriptions, instead of a list of plans. */}
              {profile.coach_id ? (
                <>
                  <div className={joinClassNames(styles.stat, styles.statCoach)}>
                    <dt>
                      <ClipboardList aria-hidden="true" /> {text.plansCount}
                    </dt>
                    <dd>{formatNumber(coach?.coach.plans_count ?? 0)}</dd>
                  </div>
                  <div className={joinClassNames(styles.stat, styles.statCoach)}>
                    <dt>
                      <BadgeCheck aria-hidden="true" /> {text.subscriptions}
                    </dt>
                    <dd>{formatNumber(coach?.coach.subscriptions ?? 0)}</dd>
                  </div>
                </>
              ) : null}
              {typeof profile.stats.views === "number" ? (
                <div className={joinClassNames(styles.stat, styles.statCoach)}>
                  <dt>
                    <Eye aria-hidden="true" /> {text.views}
                  </dt>
                  <dd>{formatNumber(profile.stats.views)}</dd>
                </div>
              ) : null}
            </dl>

            {list ? (
              <section className={styles.people} aria-label={text[list.kind]}>
                <h2 className={styles.peopleTitle}>{text[list.kind]}</h2>
                {list.people === null ? <p className={styles.muted}>{text.loading}</p> : null}
                {list.people && list.people.length === 0 ? <p className={styles.muted}>{text.empty[list.kind]}</p> : null}
                {list.people && list.people.length > 0 ? (
                  <ul className={styles.peopleList}>
                    {list.people.map((person) => (
                      <li key={person.username}>
                        <Link className={styles.person} to={profilePath(person.username)}>
                          <Avatar person={person} />
                          <span className={styles.personText}>
                            <strong>{person.nickname?.trim() || person.username}</strong>
                            {person.nickname?.trim() ? <span>@{person.username}</span> : null}
                          </span>
                        </Link>
                      </li>
                    ))}
                  </ul>
                ) : null}
              </section>
            ) : null}
          </>
        ) : null}
      </Container>
    </PageWrapper>
  );
};
