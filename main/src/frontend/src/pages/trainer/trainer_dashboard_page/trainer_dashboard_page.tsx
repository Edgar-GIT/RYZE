import {
  ArrowRight,
  ChevronRight,
  ClipboardList,
  Dumbbell,
  Eye,
  FileEdit,
  Globe2,
  Plus,
  RefreshCw,
  UserPlus,
  Users
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import { Button } from "@/components/button/button";
import { Waves } from "@/components/public_ui/public_ui";
import { type Copy, useI18n } from "@/i18n/i18n";
import { fetchCoachProfile } from "@/services/coaches_api";
import {
  fetchTrainerPremiumLevel2Requests,
  type PremiumLevel2RequestStatus,
  type StaffPremiumLevel2Request
} from "@/services/premium_level_2_api";
import {
  fetchTrainerClients,
  fetchTrainerPrograms,
  fetchTrainerProfile,
  TrainerProgramStatus,
  type TrainerClient,
  type TrainerProfile,
  type TrainerProgram
} from "@/services/trainer_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./trainer_dashboard_page.module.css";

interface DashboardData {
  profile: TrainerProfile;
  clients: TrainerClient[];
  clientTotal: number;
  programs: TrainerProgram[];
  programTotal: number;
  requests: StaffPremiumLevel2Request[];
  requestTotal: number;
  views: number | null;
}

type State = { status: "loading" } | { status: "error" } | { status: "ready"; data: DashboardData };

const COPY: Copy<{
  kicker: string;
  hello: string;
  lead: string;
  heroKicker: string;
  heroTitle: (count: number) => string;
  heroText: string;
  addClient: string;
  newProgram: string;
  publicProfile: string;
  views: (count: number) => string;
  notPublic: string;
  openProfile: string;
  clients: string;
  published: string;
  drafts: string;
  premium: string;
  recentClients: string;
  noClients: string;
  since: (date: string) => string;
  programs: string;
  noPrograms: string;
  free: string;
  draft: string;
  live: string;
  queue: string;
  noRequests: string;
  seeAll: string;
  status: Record<PremiumLevel2RequestStatus, string>;
  loadError: string;
  retry: string;
}> = {
  en: {
    kicker: "Trainer dashboard",
    hello: "Hello",
    lead: "Your clients, programmes and requests at a glance.",
    heroKicker: "Your clients",
    heroTitle: (count) => (count === 1 ? "1 client trains with you." : `${count} clients train with you.`),
    heroText: "Add the people you coach, assign them a programme and follow every workout they log.",
    addClient: "Add a client",
    newProgram: "New programme",
    publicProfile: "Public profile",
    views: (count) => (count === 1 ? "1 view of your programmes" : `${count} views of your programmes`),
    notPublic: "Publish a programme and it appears in Training plans, with your name linking to your profile.",
    openProfile: "Open public profile",
    clients: "Clients",
    published: "Published",
    drafts: "Drafts",
    premium: "Premium L2 requests",
    recentClients: "Recent clients",
    noClients: "No clients yet. Add one by their account email.",
    since: (date) => `Since ${date}`,
    programs: "Programmes",
    noPrograms: "No programmes yet.",
    free: "Free",
    draft: "Draft",
    live: "Published",
    queue: "Premium Level 2 requests",
    noRequests: "No requests allocated to you.",
    seeAll: "See all",
    status: {
      awaiting_intake: "Awaiting intake",
      awaiting_trainer_assignment: "Awaiting allocation",
      in_progress: "In progress",
      ready_for_delivery: "Ready for delivery",
      delivered: "Delivered",
      cancelled: "Cancelled"
    },
    loadError: "Unable to load your dashboard right now.",
    retry: "Try again"
  },
  pt: {
    kicker: "Painel de treinador",
    hello: "Olá",
    lead: "Os teus clientes, programas e pedidos num relance.",
    heroKicker: "Os teus clientes",
    heroTitle: (count) => (count === 1 ? "1 cliente treina contigo." : `${count} clientes treinam contigo.`),
    heroText: "Adiciona as pessoas que treinas, atribui-lhes um programa e acompanha cada treino que registam.",
    addClient: "Adicionar cliente",
    newProgram: "Novo programa",
    publicProfile: "Perfil público",
    views: (count) => (count === 1 ? "1 visualização dos teus programas" : `${count} visualizações dos teus programas`),
    notPublic: "Publica um programa e ele aparece nos Planos de treino, com o teu nome a ligar ao teu perfil.",
    openProfile: "Abrir perfil público",
    clients: "Clientes",
    published: "Publicados",
    drafts: "Rascunhos",
    premium: "Pedidos Premium N2",
    recentClients: "Clientes recentes",
    noClients: "Ainda sem clientes. Adiciona um pelo email da conta.",
    since: (date) => `Desde ${date}`,
    programs: "Programas",
    noPrograms: "Ainda sem programas.",
    free: "Grátis",
    draft: "Rascunho",
    live: "Publicado",
    queue: "Pedidos Premium Nível 2",
    noRequests: "Nenhum pedido atribuído a ti.",
    seeAll: "Ver todos",
    status: {
      awaiting_intake: "À espera do questionário",
      awaiting_trainer_assignment: "À espera de atribuição",
      in_progress: "Em curso",
      ready_for_delivery: "Pronto para entregar",
      delivered: "Entregue",
      cancelled: "Cancelado"
    },
    loadError: "Não foi possível carregar o teu painel.",
    retry: "Tentar novamente"
  }
};

const initials = (first: string, last: string, fallback: string) =>
  `${first[0] ?? ""}${last[0] ?? ""}`.toUpperCase() || fallback[0]?.toUpperCase() || "?";

// TrainerDashboardPage is the trainer's home: clients first, then the
// programmes, the public profile reach and the Premium Level 2 queue. Every
// number comes from the trainer endpoints; nothing is estimated.
export const TrainerDashboardPage = () => {
  const { locale, formatDate, formatNumber } = useI18n();
  const text = COPY[locale];
  const [state, setState] = useState<State>({ status: "loading" });
  const [retry, setRetry] = useState(0);

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const [profile, clients, programs, requests] = await Promise.all([
        fetchTrainerProfile(),
        fetchTrainerClients(1, 6),
        fetchTrainerPrograms(1, 50),
        fetchTrainerPremiumLevel2Requests(1, 5).catch(() => null)
      ]);
      const views = await fetchCoachProfile(profile.id)
        .then((coach) => coach.coach.total_views)
        .catch(() => null);
      setState({
        status: "ready",
        data: {
          profile,
          clients: clients.clients,
          clientTotal: clients.pagination.total,
          programs: programs.programs,
          programTotal: programs.pagination.total,
          requests: requests?.requests ?? [],
          requestTotal: requests?.pagination.total ?? 0,
          views
        }
      });
    } catch {
      setState({ status: "error" });
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load, retry]);

  if (state.status === "loading") return <AppLoading />;

  if (state.status === "error") {
    return (
      <section className={styles.stateCard}>
        <p>{text.loadError}</p>
        <Button variant="secondary" size="small" icon={<RefreshCw size={15} />} onClick={() => setRetry((value) => value + 1)}>
          {text.retry}
        </Button>
      </section>
    );
  }

  const { profile, clients, clientTotal, programs, programTotal, requests, requestTotal, views } = state.data;
  const published = programs.filter((program) => program.status === TrainerProgramStatus.PUBLISHED).length;
  const drafts = programs.length - published;
  const firstName = profile.user.first_name;

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <p className={styles.kicker}>{text.kicker}</p>
        <h1 className={styles.title}>
          {text.hello}
          {firstName ? (
            <>
              , <span>{firstName}</span>
            </>
          ) : null}
          .
        </h1>
        <p className={styles.lead}>{text.lead}</p>
      </header>

      <div className={styles.grid}>
        <section className={styles.hero} aria-labelledby="trainer-hero">
          <p className={styles.heroKicker}>
            <Users aria-hidden="true" /> {text.heroKicker}
          </p>
          <h2 id="trainer-hero" className={styles.heroTitle}>
            {text.heroTitle(clientTotal)}
          </h2>
          <p className={styles.heroText}>{text.heroText}</p>
          <div className={styles.heroActions}>
            <Link className={styles.inkButton} to="/trainer/clients">
              <UserPlus aria-hidden="true" /> {text.addClient}
            </Link>
            <Link className={styles.ghostInk} to="/trainer/programs/new">
              <Plus aria-hidden="true" /> {text.newProgram}
            </Link>
          </div>
          <Waves className={styles.heroWaves} />
        </section>

        <section className={joinClassNames(styles.card, styles.reach)} aria-labelledby="trainer-reach">
          <h2 id="trainer-reach" className={styles.cardTitle}>
            <Globe2 aria-hidden="true" /> {text.publicProfile}
          </h2>
          {views !== null ? (
            <>
              <p className={styles.bigNumber}>
                <Eye aria-hidden="true" />
                {formatNumber(views)}
              </p>
              <p className={styles.muted}>{text.views(views)}</p>
              <Link className={styles.cardLink} to={`/coaches/${encodeURIComponent(profile.id)}`}>
                {text.openProfile} <ArrowRight aria-hidden="true" />
              </Link>
            </>
          ) : (
            <p className={styles.muted}>{text.notPublic}</p>
          )}
        </section>

        <section className={styles.tiles} aria-label={text.kicker}>
          {[
            { icon: Users, label: text.clients, value: clientTotal, to: "/trainer/clients" },
            { icon: Dumbbell, label: text.published, value: published, to: "/trainer/programs" },
            { icon: FileEdit, label: text.drafts, value: drafts, to: "/trainer/programs" },
            { icon: ClipboardList, label: text.premium, value: requestTotal, to: "/trainer/premium-level-2" }
          ].map(({ icon: Icon, label, value, to }) => (
            <Link key={label} className={styles.tile} to={to}>
              <span className={styles.tileIcon} aria-hidden="true">
                <Icon />
              </span>
              <span className={styles.tileLabel}>{label}</span>
              <strong className={styles.tileValue}>{formatNumber(value)}</strong>
            </Link>
          ))}
        </section>

        <section className={joinClassNames(styles.card, styles.half)} aria-labelledby="trainer-clients">
          <div className={styles.cardHead}>
            <h2 id="trainer-clients" className={styles.cardTitle}>
              {text.recentClients}
            </h2>
            <Link className={styles.cardLink} to="/trainer/clients">
              {text.seeAll} <ChevronRight aria-hidden="true" />
            </Link>
          </div>
          {clients.length === 0 ? (
            <p className={styles.muted}>{text.noClients}</p>
          ) : (
            <ul className={styles.list}>
              {clients.map((client) => (
                <li key={client.id}>
                  <Link className={styles.row} to={`/trainer/clients/${encodeURIComponent(client.user_id)}`}>
                    <span className={styles.avatar} aria-hidden="true">
                      {initials(client.user.first_name, client.user.last_name, client.user.email)}
                    </span>
                    <span className={styles.rowText}>
                      <strong>{[client.user.first_name, client.user.last_name].filter(Boolean).join(" ") || client.user.email}</strong>
                      <span>{client.user.email}</span>
                    </span>
                    <span className={styles.rowMeta}>{text.since(formatDate(client.created_at, { day: "numeric", month: "short" }))}</span>
                    <ChevronRight className={styles.chevron} aria-hidden="true" />
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className={joinClassNames(styles.card, styles.half)} aria-labelledby="trainer-programs">
          <div className={styles.cardHead}>
            <h2 id="trainer-programs" className={styles.cardTitle}>
              {text.programs} · {formatNumber(programTotal)}
            </h2>
            <Link className={styles.cardLink} to="/trainer/programs">
              {text.seeAll} <ChevronRight aria-hidden="true" />
            </Link>
          </div>
          {programs.length === 0 ? (
            <p className={styles.muted}>{text.noPrograms}</p>
          ) : (
            <ul className={styles.list}>
              {programs.slice(0, 6).map((program) => (
                <li key={program.id}>
                  <Link className={styles.row} to={`/trainer/programs/${encodeURIComponent(program.id)}`}>
                    <span className={joinClassNames(styles.avatar, styles.programIcon)} aria-hidden="true">
                      <Dumbbell />
                    </span>
                    <span className={styles.rowText}>
                      <strong>{program.name}</strong>
                      <span>
                        {program.type === "free"
                          ? text.free
                          : new Intl.NumberFormat(locale === "pt" ? "pt-PT" : "en-GB", { style: "currency", currency: program.currency || "EUR" }).format(program.price_minor_units / 100)}
                      </span>
                    </span>
                    <span
                      className={joinClassNames(
                        styles.status,
                        program.status === TrainerProgramStatus.PUBLISHED ? styles.statusLive : styles.statusDraft
                      )}
                    >
                      {program.status === TrainerProgramStatus.PUBLISHED ? text.live : text.draft}
                    </span>
                    <ChevronRight className={styles.chevron} aria-hidden="true" />
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className={joinClassNames(styles.card, styles.full)} aria-labelledby="trainer-queue">
          <div className={styles.cardHead}>
            <h2 id="trainer-queue" className={styles.cardTitle}>
              <ClipboardList aria-hidden="true" /> {text.queue}
            </h2>
            <Link className={styles.cardLink} to="/trainer/premium-level-2">
              {text.seeAll} <ChevronRight aria-hidden="true" />
            </Link>
          </div>
          {requests.length === 0 ? (
            <p className={styles.muted}>{text.noRequests}</p>
          ) : (
            <ul className={styles.list}>
              {requests.map((request) => (
                <li key={request.id}>
                  <Link className={styles.row} to={`/trainer/premium-level-2/${encodeURIComponent(request.id)}`}>
                    <span className={styles.avatar} aria-hidden="true">
                      {initials(request.client_first_name, request.client_last_name, request.client_email)}
                    </span>
                    <span className={styles.rowText}>
                      <strong>{[request.client_first_name, request.client_last_name].filter(Boolean).join(" ") || request.client_email}</strong>
                      <span>{request.offer_name}</span>
                    </span>
                    <span className={joinClassNames(styles.status, request.status === "delivered" ? styles.statusLive : styles.statusDraft)}>
                      {text.status[request.status] ?? request.status}
                    </span>
                    <ChevronRight className={styles.chevron} aria-hidden="true" />
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </div>
  );
};
