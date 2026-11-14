import {
  ClipboardList,
  Dumbbell,
  ExternalLink,
  Globe2,
  LayoutDashboard,
  LifeBuoy,
  Plus,
  Settings,
  UserRound,
  Users
} from "lucide-react";
import { useEffect, useState } from "react";
import { Redirect, useHistory } from "react-router-dom";

import { AppShell, type ShellNavItem, type UserMenuItem } from "@/components/app_shell/app_shell";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { useTrainerSession } from "@/components/trainer/trainer_session_context";
import { type Copy, useI18n } from "@/i18n/i18n";
import { fetchTrainerProfile } from "@/services/trainer_api";
import { apiPost } from "@utils/http_client";

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
  support: string;
  newProgram: string;
  newProgramText: string;
  memberArea: string;
  back: string;
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
    premium: "Premium L2",
    programs: "Programs",
    more: "More",
    profile: "Profile",
    support: "Support",
    newProgram: "New programme",
    newProgramText: "Build a plan for clients or the store",
    memberArea: "My training",
    back: "Back to the site",
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
    premium: "Premium N2",
    programs: "Programas",
    more: "Mais",
    profile: "Perfil",
    support: "Suporte",
    newProgram: "Novo programa",
    newProgramText: "Cria um plano para clientes ou para a loja",
    memberArea: "O meu treino",
    back: "Voltar ao site",
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
  const [user, setUser] = useState<{ name: string; email: string } | null>(null);
  const [profileId, setProfileId] = useState<string | null>(null);

  const isTrainer = access?.is_trainer ?? false;

  useEffect(() => {
    if (!isTrainer) return;
    let cancelled = false;
    fetchTrainerProfile()
      .then((profile) => {
        if (!cancelled) {
          const name = [profile.user.first_name, profile.user.last_name].filter(Boolean).join(" ");
          setUser({ name: name || profile.user.email, email: profile.user.email });
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

  const items: ShellNavItem[] = [
    { to: "/trainer/dashboard", label: text.dashboard, icon: LayoutDashboard },
    { to: "/trainer/clients", label: text.clients, icon: Users },
    { to: "/trainer/premium-level-2", label: text.premium, icon: ClipboardList },
    { to: "/trainer/programs", label: text.programs, icon: Dumbbell }
  ];
  const secondaryItems: ShellNavItem[] = [
    { to: "/trainer/profile", label: text.profile, icon: UserRound },
    { to: "/support", label: text.support, icon: LifeBuoy }
  ];
  const menuItems: UserMenuItem[] = [
    { to: "/trainer/profile", label: text.profile, icon: UserRound },
    ...(profileId ? [{ to: `/coaches/${profileId}`, label: text.publicProfile, icon: ExternalLink }] : []),
    { to: "/profile", label: text.memberArea, icon: Dumbbell },
    { to: "/account/settings", label: text.account, icon: Settings },
    { to: "/support", label: text.support, icon: LifeBuoy },
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
      groups={[{ items }, { label: text.more, items: secondaryItems }]}
      tabs={items}
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
    >
      {children}
    </AppShell>
  );
};
