import { Archive, Eye, FileText, RefreshCw, ShoppingBag, Users, Wallet } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import { type Copy, useI18n } from "@/i18n/i18n";
import { fetchCoachProfile, type CoachProgram } from "@/services/coaches_api";
import { fetchTrainerPremiumLevel2Requests } from "@/services/premium_level_2_api";
import {
  fetchTrainerClients,
  fetchTrainerEarnings,
  fetchTrainerProfile,
  fetchTrainerPrograms,
  TrainerProgramStatus,
  TrainerProgramType,
  type TrainerEarningsTotals,
  type TrainerProgram
} from "@/services/trainer_api";

import styles from "./trainer_stats_page.module.css";

const COPY: Copy<{
  kicker: string;
  title: string;
  lead: string;
  earnings: string;
  earningsLead: string;
  sales: string;
  gross: string;
  commission: string;
  net: string;
  noSales: string;
  views: string;
  viewsLead: string;
  totalViews: string;
  noViews: string;
  viewsOf: (count: string) => string;
  programs: string;
  programsLead: string;
  onSale: string;
  drafts: string;
  archived: string;
  free: string;
  premium: string;
  managePrograms: string;
  people: string;
  peopleLead: string;
  clients: string;
  requests: string;
  openClients: string;
  openRequests: string;
  error: string;
  retry: string;
}> = {
  en: {
    kicker: "Trainer · statistics",
    title: "Statistics and earnings",
    lead: "How your plans are doing: what you sold and earned, how many people opened each plan, and where your programmes and clients stand.",
    earnings: "Sales and earnings",
    earningsLead: "Paid purchases of your Premium plans and Level 2 work. RYZE keeps its commission; the net amount is yours.",
    sales: "Sales",
    gross: "Sold (gross)",
    commission: "RYZE commission",
    net: "Your earnings (net)",
    noSales: "No sales yet. Publish a Premium package to start selling.",
    views: "Views of your plans",
    viewsLead: "How many times people opened each of your published plans in the store.",
    totalViews: "Total views",
    noViews: "Your published plans have not been opened yet.",
    viewsOf: (count) => `${count} views`,
    programs: "Your programmes",
    programsLead: "Every plan you created, by state and type.",
    onSale: "On sale",
    drafts: "Drafts",
    archived: "Archived",
    free: "Free",
    premium: "Premium (Level 1)",
    managePrograms: "Manage my programmes",
    people: "Clients and requests",
    peopleLead: "People who have one of your plans, and the Premium Level 2 requests assigned to you.",
    clients: "Clients",
    requests: "Level 2 requests",
    openClients: "See my clients",
    openRequests: "See the requests",
    error: "The statistics could not be loaded.",
    retry: "Try again"
  },
  pt: {
    kicker: "Treinador · estatísticas",
    title: "Estatísticas e ganhos",
    lead: "Como estão a correr os teus planos: o que vendeste e ganhaste, quantas pessoas abriram cada plano e em que ponto estão os teus programas e clientes.",
    earnings: "Vendas e ganhos",
    earningsLead: "Compras pagas dos teus planos Premium e do trabalho no Nível 2. A RYZE fica com a comissão; o valor líquido é teu.",
    sales: "Vendas",
    gross: "Vendido (bruto)",
    commission: "Comissão RYZE",
    net: "Os teus ganhos (líquido)",
    noSales: "Ainda sem vendas. Publica um pacote Premium para começares a vender.",
    views: "Visualizações dos teus planos",
    viewsLead: "Quantas vezes as pessoas abriram cada um dos teus planos publicados na loja.",
    totalViews: "Total de visualizações",
    noViews: "Os teus planos publicados ainda não foram abertos.",
    viewsOf: (count) => `${count} visualizações`,
    programs: "Os teus programas",
    programsLead: "Todos os planos que criaste, por estado e por tipo.",
    onSale: "À venda",
    drafts: "Rascunhos",
    archived: "Arquivados",
    free: "Grátis",
    premium: "Premium (Nível 1)",
    managePrograms: "Gerir os meus programas",
    people: "Clientes e pedidos",
    peopleLead: "Pessoas que têm um dos teus planos e os pedidos Premium Nível 2 atribuídos a ti.",
    clients: "Clientes",
    requests: "Pedidos Nível 2",
    openClients: "Ver os meus clientes",
    openRequests: "Ver os pedidos",
    error: "Não foi possível carregar as estatísticas.",
    retry: "Tentar de novo"
  }
};

interface Data {
  earnings: TrainerEarningsTotals | null;
  totalViews: number;
  viewed: CoachProgram[];
  programs: TrainerProgram[];
  clients: number;
  requests: number;
}

// TrainerStatsPage gathers the trainer's numbers on one page (owner request,
// 2026-11-27): earnings, plan views, programmes and clients, each explained.
export const TrainerStatsPage = () => {
  const { locale, formatMoney, formatNumber } = useI18n();
  const text = COPY[locale];
  const [data, setData] = useState<Data | null>(null);
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");

  const load = useCallback(async () => {
    setStatus("loading");
    try {
      const [profile, programs, clients, requests, earnings] = await Promise.all([
        fetchTrainerProfile(),
        fetchTrainerPrograms(1, 100),
        fetchTrainerClients(1, 1),
        fetchTrainerPremiumLevel2Requests(1, 1).catch(() => null),
        fetchTrainerEarnings()
          .then((report) => report.totals[0] ?? null)
          .catch(() => null)
      ]);
      const coach = await fetchCoachProfile(profile.id).catch(() => null);
      setData({
        earnings,
        totalViews: coach?.coach.total_views ?? 0,
        viewed: [...(coach?.programs ?? [])].sort((a, b) => b.views - a.views),
        programs: programs.programs,
        clients: clients.pagination.total,
        requests: requests?.pagination.total ?? 0
      });
      setStatus("ready");
    } catch {
      setStatus("error");
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (status === "loading") return <AppLoading />;
  if (status === "error" || !data) {
    return (
      <section className={styles.state}>
        <p>{text.error}</p>
        <button type="button" className={styles.link} onClick={() => void load()}>
          <RefreshCw aria-hidden="true" /> {text.retry}
        </button>
      </section>
    );
  }

  const count = (test: (program: TrainerProgram) => boolean) => data.programs.filter(test).length;
  const topViews = data.viewed[0]?.views ?? 0;

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <p className={styles.kicker}>{text.kicker}</p>
        <h1 className={styles.title}>{text.title}</h1>
        <p className={styles.lead}>{text.lead}</p>
      </header>

      <section className={styles.card} aria-labelledby="stats-earnings">
        <h2 id="stats-earnings" className={styles.cardTitle}>
          <Wallet aria-hidden="true" /> {text.earnings}
        </h2>
        <p className={styles.cardLead}>{text.earningsLead}</p>
        {data.earnings && data.earnings.sales > 0 ? (
          <dl className={styles.figures}>
            <div>
              <dt>{text.sales}</dt>
              <dd>{formatNumber(data.earnings.sales)}</dd>
            </div>
            <div>
              <dt>{text.gross}</dt>
              <dd>{formatMoney(data.earnings.gross, data.earnings.currency)}</dd>
            </div>
            <div>
              <dt>{text.commission}</dt>
              <dd>{formatMoney(data.earnings.commission, data.earnings.currency)}</dd>
            </div>
            <div className={styles.highlight}>
              <dt>{text.net}</dt>
              <dd>{formatMoney(data.earnings.net, data.earnings.currency)}</dd>
            </div>
          </dl>
        ) : (
          <p className={styles.empty}>{text.noSales}</p>
        )}
      </section>

      <section className={styles.card} aria-labelledby="stats-views">
        <h2 id="stats-views" className={styles.cardTitle}>
          <Eye aria-hidden="true" /> {text.views}
        </h2>
        <p className={styles.cardLead}>{text.viewsLead}</p>
        <p className={styles.big}>
          {formatNumber(data.totalViews)} <span>{text.totalViews}</span>
        </p>
        {data.viewed.length > 0 && topViews > 0 ? (
          <ul className={styles.bars}>
            {data.viewed.map((program) => (
              <li key={program.id}>
                <span className={styles.barLabel}>
                  <Link to={`/trainer/programs/${program.id}`}>{program.name}</Link>
                  <em>{text.viewsOf(formatNumber(program.views))}</em>
                </span>
                <span className={styles.barTrack} aria-hidden="true">
                  <span className={styles.barFill} style={{ width: `${Math.max(2, (program.views / topViews) * 100)}%` }} />
                </span>
              </li>
            ))}
          </ul>
        ) : (
          <p className={styles.empty}>{text.noViews}</p>
        )}
      </section>

      <div className={styles.split}>
        <section className={styles.card} aria-labelledby="stats-programs">
          <h2 id="stats-programs" className={styles.cardTitle}>
            <FileText aria-hidden="true" /> {text.programs}
          </h2>
          <p className={styles.cardLead}>{text.programsLead}</p>
          <dl className={styles.figures}>
            <div>
              <dt>{text.onSale}</dt>
              <dd>{count((p) => p.status === TrainerProgramStatus.PUBLISHED && !p.archived_at)}</dd>
            </div>
            <div>
              <dt>{text.drafts}</dt>
              <dd>{count((p) => p.status !== TrainerProgramStatus.PUBLISHED)}</dd>
            </div>
            <div>
              <dt>
                <Archive aria-hidden="true" /> {text.archived}
              </dt>
              <dd>{count((p) => Boolean(p.archived_at))}</dd>
            </div>
            <div>
              <dt>{text.free}</dt>
              <dd>{count((p) => p.type === TrainerProgramType.FREE)}</dd>
            </div>
            <div>
              <dt>{text.premium}</dt>
              <dd>{count((p) => p.type === TrainerProgramType.PREMIUM)}</dd>
            </div>
          </dl>
          <Link className={styles.link} to="/trainer/programs">
            {text.managePrograms}
          </Link>
        </section>

        <section className={styles.card} aria-labelledby="stats-people">
          <h2 id="stats-people" className={styles.cardTitle}>
            <Users aria-hidden="true" /> {text.people}
          </h2>
          <p className={styles.cardLead}>{text.peopleLead}</p>
          <dl className={styles.figures}>
            <div>
              <dt>{text.clients}</dt>
              <dd>{formatNumber(data.clients)}</dd>
            </div>
            <div>
              <dt>
                <ShoppingBag aria-hidden="true" /> {text.requests}
              </dt>
              <dd>{formatNumber(data.requests)}</dd>
            </div>
          </dl>
          <div className={styles.links}>
            <Link className={styles.link} to="/trainer/clients">
              {text.openClients}
            </Link>
            <Link className={styles.link} to="/trainer/premium-level-2">
              {text.openRequests}
            </Link>
          </div>
        </section>
      </div>
    </div>
  );
};
