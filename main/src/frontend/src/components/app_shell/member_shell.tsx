import {
  BarChart3,
  BookOpen,
  Dumbbell,
  History,
  Globe2,
  LayoutDashboard,
  LifeBuoy,
  Play,
  ReceiptText,
  Settings,
  ShoppingBag,
  Users
} from "lucide-react";
import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { useHistory } from "react-router-dom";

import { type Copy, useI18n } from "@/i18n/i18n";
import { fetchTrainerAccess } from "@/services/trainer_api";
import { ApiError, apiGet, apiPost } from "@utils/http_client";

import { AppShell, type ShellNavItem, type UserMenuItem } from "./app_shell";

export interface MemberUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  created_at: string;
  email_verified: boolean;
  preferred_language: string | null;
  password_set: boolean;
}

interface MemberContextValue {
  user: MemberUser | null;
  isTrainer: boolean;
  reload: () => void;
}

const MemberContext = createContext<MemberContextValue>({ user: null, isTrainer: false, reload: () => undefined });

// The signed-in member, shared by every page of the member area.
export const useMember = () => useContext(MemberContext);

const COPY: Copy<{
  nav: string;
  home: string;
  train: string;
  progress: string;
  history: string;
  library: string;
  plans: string;
  purchases: string;
  support: string;
  account: string;
  more: string;
  menu: string;
  signOut: string;
  signingOut: string;
  site: string;
  member: string;
  close: string;
  start: string;
  startText: string;
  trainer: string;
  back: string;
}> = {
  en: {
    nav: "Member area",
    home: "Home",
    train: "Train",
    progress: "Progress",
    history: "History",
    library: "Exercises",
    plans: "Training plans",
    purchases: "Purchases",
    support: "Support",
    account: "Account",
    more: "More",
    menu: "Account menu",
    signOut: "Sign out",
    signingOut: "Signing out…",
    site: "RYZE website",
    member: "Member",
    close: "Close",
    start: "Start a workout",
    startText: "Pick up your plan where you left it",
    trainer: "Trainer area",
    back: "Back to the site"
  },
  pt: {
    nav: "Área de membro",
    home: "Início",
    train: "Treinar",
    progress: "Progresso",
    history: "Histórico",
    library: "Exercícios",
    plans: "Planos de treino",
    purchases: "Compras",
    support: "Suporte",
    account: "Conta",
    more: "Mais",
    menu: "Menu da conta",
    signOut: "Terminar sessão",
    signingOut: "A terminar sessão…",
    site: "Site da RYZE",
    member: "Membro",
    close: "Fechar",
    start: "Começar treino",
    startText: "Continua o teu plano onde ficaste",
    trainer: "Área de treinador",
    back: "Voltar ao site"
  }
};

const isTrainPath = (pathname: string) =>
  pathname.startsWith("/services/my-programs") || pathname.startsWith("/services/workout-sessions");

// MemberShell frames every page of the member area: it loads the account once
// (a visitor without a session goes to the login page) and renders the app
// shell around the pages.
export const MemberShell = ({ children }: { children: ReactNode }) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const history = useHistory();
  const [user, setUser] = useState<MemberUser | null>(null);
  const [isTrainer, setIsTrainer] = useState(false);
  const [version, setVersion] = useState(0);

  useEffect(() => {
    let cancelled = false;
    apiGet<MemberUser>("/me")
      .then((me) => {
        if (!cancelled) setUser(me);
      })
      .catch((error) => {
        if (!cancelled && error instanceof ApiError && error.status === 401) {
          history.replace("/login");
        }
      });
    fetchTrainerAccess()
      .then((access) => {
        if (!cancelled) setIsTrainer(access.is_trainer);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [history, version]);

  const items: ShellNavItem[] = [
    { to: "/profile", label: text.home, icon: LayoutDashboard, match: (pathname) => pathname === "/profile" },
    { to: "/services/my-programs", label: text.train, icon: Dumbbell, match: isTrainPath },
    { to: "/services/progress", label: text.progress, icon: BarChart3 },
    { to: "/services/workout-history", label: text.history, icon: History }
  ];
  const secondaryItems: ShellNavItem[] = [
    { to: "/services/generic-program", label: text.plans, icon: ShoppingBag },
    { to: "/exercises", label: text.library, icon: BookOpen },
    { to: "/account/purchases", label: text.purchases, icon: ReceiptText },
    { to: "/support", label: text.support, icon: LifeBuoy },
    { to: "/account/settings", label: text.account, icon: Settings }
  ];
  const menuItems: UserMenuItem[] = [
    { to: "/account/settings", label: text.account, icon: Settings },
    { to: "/account/purchases", label: text.purchases, icon: ReceiptText },
    { to: "/support", label: text.support, icon: LifeBuoy },
    ...(isTrainer ? [{ to: "/trainer/dashboard", label: text.trainer, icon: Users }] : []),
    { to: "/", label: text.site, icon: Globe2 }
  ];

  // Sign out ends the session on the server, then leaves for the login page;
  // a failed request still leaves (the session cookie may already be gone).
  const signOut = async () => {
    try {
      await apiPost("/auth/logout", {});
    } catch {
      // Best effort.
    }
    history.replace("/login");
  };

  const name = user ? `${user.first_name} ${user.last_name}`.trim() || user.email : "";

  return (
    <MemberContext.Provider value={{ user, isTrainer, reload: () => setVersion((value) => value + 1) }}>
      <AppShell
        area="member"
        navLabel={text.nav}
        groups={[{ items }, { label: text.more, items: secondaryItems }]}
        tabs={items}
        moreLabel={text.more}
        closeLabel={text.close}
        user={user ? { name, email: user.email, role: isTrainer ? undefined : text.member } : null}
        menu={{
          label: text.menu,
          items: menuItems,
          signOutLabel: text.signOut,
          signingOutLabel: text.signingOut,
          onSignOut: signOut
        }}
        action={{ to: "/services/my-programs", title: text.start, text: text.startText, icon: Play }}
        switchTo={isTrainer ? { to: "/trainer/dashboard", label: text.trainer, icon: Users } : undefined}
        backToSite={text.back}
      >
        {children}
      </AppShell>
    </MemberContext.Provider>
  );
};
