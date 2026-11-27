import {
  Archive,
  BarChart3,
  ClipboardList,
  Dumbbell,
  ExternalLink,
  FilePlus2,
  Globe2,
  LayoutDashboard,
  Library,
  Plus,
  Settings,
  UserRound,
  Users
} from "lucide-react";
import { useEffect, useState } from "react";
import { Redirect, useHistory, useLocation } from "react-router-dom";

import { AppShell, type ShellNavItem, type UserMenuItem } from "@/components/app_shell/app_shell";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { useTrainerSession } from "@/components/trainer/trainer_session_context";
import { type Copy, useI18n } from "@/i18n/i18n";
import { fetchTrainerProfile } from "@/services/trainer_api";
import type { MemberUser } from "@/components/app_shell/member_shell";
import { publicNameOf } from "@/services/profile_api";
import { apiGet, apiPost, publicFileUrl } from "@utils/http_client";

interface TrainerLayoutProps {
  children: React.ReactNode;
}

const COPY: Copy<{
  nav: string;
  dashboard: string;
  clients: string;
  premium: string;
  programs: string;
  more: string;
  profile: string;
  groupWork: string;
  groupPrograms: string;
  groupProfile: string;
  dashboardLong: string;
  clientsLong: string;
  premiumLong: string;
  stats: string;
  programsLong: string;
  createProgram: string;
  archive: string;
  profileLong: string;
  viewPublic: string;
  newProgram: string;
  newProgramText: string;
  memberArea: string;
  back: string;
  goBack: string;
  menu: string;
  publicProfile: string;
  account: string;
  site: string;
  signOut: string;
  signingOut: string;
  coach: string;
  close: string;
}> = {
  en: {
    nav: "Trainer area",
    dashboard: "Dashboard",
    clients: "Clients",
    premium: "Level 2",
    programs: "Programmes",
    more: "More",
    profile: "Profile",
    groupWork: "My work",
    groupPrograms: "Programmes",
    groupProfile: "Profile and account",
    dashboardLong: "Overview of today",
    clientsLong: "My clients",
    premiumLong: "Premium Level 2 requests",
    stats: "Statistics and earnings",
    programsLong: "My programmes",
    createProgram: "Create a programme",
    archive: "Archived programmes",
    profileLong: "Edit my trainer profile",
    viewPublic: "See my public profile",
    newProgram: "New programme",
    newProgramText: "Build a plan for clients or the store",
    memberArea: "My training",
    back: "Back to the site",
    goBack: "Back",
    menu: "Account menu",
    publicProfile: "Public coach profile",
    account: "Account settings",
    site: "RYZE website",
    signOut: "Sign out",
    signingOut: "Signing out…",
    coach: "Coach",
    close: "Close"
  },
  pt: {
    nav: "Área de treinador",
    dashboard: "Painel",
    clients: "Clientes",
    premium: "Nível 2",
    programs: "Programas",
    more: "Mais",
    profile: "Perfil",
    groupWork: "O meu trabalho",
    groupPrograms: "Programas",
    groupProfile: "Perfil e conta",
    dashboardLong: "Resumo do dia",
    clientsLong: "Os meus clientes",
    premiumLong: "Pedidos Premium Nível 2",
    stats: "Estatísticas e ganhos",
    programsLong: "Os meus programas",
    createProgram: "Criar um programa",
    archive: "Programas arquivados",
    profileLong: "Editar o meu perfil de treinador",
    viewPublic: "Ver o meu perfil público",
    newProgram: "Novo programa",
    newProgramText: "Cria um plano para clientes ou para a loja",
    memberArea: "O meu treino",
    back: "Voltar ao site",
    goBack: "Voltar",
    menu: "Menu da conta",
    publicProfile: "Perfil público",
    account: "Definições da conta",
    site: "Site da RYZE",
    signOut: "Terminar sessão",
    signingOut: "A terminar sessão…",
    coach: "Treinador",
    close: "Fechar"
  }
};

// TrainerLayout is the shell of the approved-trainer area. It resolves the
// trainer session first: visitors without a session go to the login page and
// members who are not trainers go to the application page, so a trainer-only
// surface never renders for someone the backend has not confirmed. The frame
// is the same app shell as the member area (sidebar, top bar, phone tabs).
export const TrainerLayout = ({ children }: TrainerLayoutProps) => {
  const { access, unauthenticated, loaded } = useTrainerSession();
  const { locale } = useI18n();
  const text = COPY[locale];
  const history = useHistory();
  const location = useLocation();
  const [user, setUser] = useState<{ name: string; email: string; handle?: string; avatarUrl?: string } | null>(null);
  const [profileId, setProfileId] = useState<string | null>(null);

  const isTrainer = access?.is_trainer ?? false;

  useEffect(() => {
    if (!isTrainer) return;
    let cancelled = false;
    // The shell shows the person's one RYZE identity: photo, nickname (else
    // the @username) and the @username, never the email as a name.
    Promise.all([fetchTrainerProfile(), apiGet<MemberUser>("/me")])
      .then(([profile, me]) => {
        if (!cancelled) {
          setUser({
            name: publicNameOf(me),
            email: me.email,
            handle: me.username,
            avatarUrl: me.avatar_url ? publicFileUrl(me.avatar_url) : undefined
          });
          setProfileId(profile.id);
        }
      })
      .catch(() => {
        // The shell renders without the name; the pages load their own data.
      });
    return () => {
      cancelled = true;
    };
  }, [isTrainer]);

  if (!loaded) {
    return <LoadingScreen />;
  }

  if (unauthenticated) {
    return <Redirect to="/login" />;
  }

  if (!isTrainer) {
    return <Redirect to="/trainer/apply" />;
  }

  // The phone tabs keep the four main sections; the sidebar spells every
  // section out (owner request, 2026-11-27: explicit, nothing to guess).
  const tabs: ShellNavItem[] = [
    { to: "/trainer/dashboard", label: text.dashboard, icon: LayoutDashboard },
    { to: "/trainer/clients", label: text.clients, icon: Users },
    { to: "/trainer/premium-level-2", label: text.premium, icon: ClipboardList },
    { to: "/trainer/programs", label: text.programs, icon: Dumbbell }
  ];
  const archiveView = new URLSearchParams(location.search).get("view") === "archived";
  const inPrograms = (pathname: string) => pathname.startsWith("/trainer/programs") && pathname !== "/trainer/programs/new";
  const workItems: ShellNavItem[] = [
    { to: "/trainer/dashboard", label: text.dashboardLong, icon: LayoutDashboard },
    { to: "/trainer/clients", label: text.clientsLong, icon: Users },
    { to: "/trainer/premium-level-2", label: text.premiumLong, icon: ClipboardList },
    { to: "/trainer/stats", label: text.stats, icon: BarChart3 }
  ];
  const programItems: ShellNavItem[] = [
    { to: "/trainer/programs", label: text.programsLong, icon: Library, match: (pathname) => inPrograms(pathname) && !archiveView },
    { to: "/trainer/programs/new", label: text.createProgram, icon: FilePlus2, match: (pathname) => pathname === "/trainer/programs/new" },
    { to: "/trainer/programs?view=archived", label: text.archive, icon: Archive, match: (pathname) => pathname === "/trainer/programs" && archiveView }
  ];
  const publicProfilePath = user?.handle ? `/u/${encodeURIComponent(user.handle)}` : profileId ? `/coaches/${profileId}` : "";
  const secondaryItems: ShellNavItem[] = [
    { to: "/trainer/profile", label: text.profileLong, icon: UserRound },
    ...(publicProfilePath ? [{ to: publicProfilePath, label: text.viewPublic, icon: ExternalLink }] : []),
    { to: "/account/settings", label: text.account, icon: Settings }
  ];
  const menuItems: UserMenuItem[] = [
    { to: "/trainer/profile", label: text.profile, icon: UserRound },
    ...(publicProfilePath ? [{ to: publicProfilePath, label: text.publicProfile, icon: ExternalLink }] : []),
    { to: "/profile", label: text.memberArea, icon: Dumbbell },
    { to: "/account/settings", label: text.account, icon: Settings },
    { to: "/", label: text.site, icon: Globe2 }
  ];

  const signOut = async () => {
    try {
      await apiPost("/auth/logout", {});
    } catch {
      // Best effort: the session cookie may already be gone.
    }
    history.replace("/login");
  };

  return (
    <AppShell
      area="trainer"
      brandTag={text.coach}
      navLabel={text.nav}
      groups={[
        { label: text.groupWork, items: workItems },
        { label: text.groupPrograms, items: programItems },
        { label: text.groupProfile, items: secondaryItems }
      ]}
      tabs={tabs}
      moreLabel={text.more}
      closeLabel={text.close}
      user={user ? { ...user, role: text.coach } : null}
      menu={{
        label: text.menu,
        items: menuItems,
        signOutLabel: text.signOut,
        signingOutLabel: text.signingOut,
        onSignOut: signOut
      }}
      action={{ to: "/trainer/programs/new", title: text.newProgram, text: text.newProgramText, icon: Plus }}
      switchTo={{ to: "/profile", label: text.memberArea, icon: Dumbbell }}
      backToSite={text.back}
      homePath="/trainer/dashboard"
      backLabel={text.goBack}
    >
      {children}
    </AppShell>
  );
};
