import {
  BarChart3,
  HeartPulse,
  BookOpen,
  Dumbbell,
  History,
  Globe2,
  GraduationCap,
  LayoutDashboard,
  Play,
  ReceiptText,
  Settings,
  ShoppingBag,
  Trophy,
  UserRound,
  Users
} from "lucide-react";
import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { useHistory } from "react-router-dom";

import { type Copy, useI18n } from "@/i18n/i18n";
import { resolveBack } from "@/services/navigation_history";
import { publicNameOf } from "@/services/profile_api";
import { profilePath } from "@/services/public_profiles_api";
import { fetchMyPremiumLevel2Requests } from "@/services/premium_level_2_api";
import { fetchTrainerAccess } from "@/services/trainer_api";
import { ApiError, apiGet, apiPost, publicFileUrl } from "@utils/http_client";

import { AppShell, type ShellNavItem, type UserMenuItem } from "./app_shell";
import { ChooseUsernameDialog } from "./choose_username_dialog";

export interface MemberUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  created_at: string;
  email_verified: boolean;
  preferred_language: string | null;
  password_set: boolean;
  sex?: "female" | "male" | null;
  plan_audience_filter?: boolean | null;
  /** The unique @username (public reference). */
  username: string;
  /** The name people see; null: the registered name. */
  display_name: string | null;
  /** Public path of the profile photo; null without one. */
  avatar_url: string | null;
  /** The @username was generated: the person is asked to choose one. */
  needs_username?: boolean;
  // When the nickname may change again (it changes once every 7 days).
  nickname_change_at?: string;
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
  recovery: string;
  achievements: string;
  history: string;
  library: string;
  plans: string;
  purchases: string;
  account: string;
  publicProfile: string;
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
  academy: string;
  back: string;
  goBack: string;
}> = {
  en: {
    nav: "Member area",
    home: "Home",
    train: "Train",
    progress: "Progress",
    recovery: "Recovery",
    achievements: "Achievements",
    history: "History",
    library: "Exercises",
    plans: "Training plans",
    purchases: "Purchases",
    account: "Account",
    publicProfile: "My public profile",
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
    academy: "Premium L2 academy",
    back: "Back to the site",
    goBack: "Back"
  },
  pt: {
    nav: "Área de membro",
    home: "Início",
    train: "Treinar",
    progress: "Progresso",
    recovery: "Recuperação",
    achievements: "Conquistas",
    history: "Histórico",
    library: "Exercícios",
    plans: "Planos de treino",
    purchases: "Compras",
    account: "Conta",
    publicProfile: "O meu perfil público",
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
    academy: "Academia Premium N2",
    back: "Voltar ao site",
    goBack: "Voltar"
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
  const [hasPremiumLevel2Academy, setHasPremiumLevel2Academy] = useState(false);
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
    fetchMyPremiumLevel2Requests()
      .then((requests) => {
        if (!cancelled) setHasPremiumLevel2Academy(requests.some((request) => request.status === "delivered"));
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
    { to: "/services/recovery", label: text.recovery, icon: HeartPulse },
    { to: "/services/workout-history", label: text.history, icon: History }
  ];
  const secondaryItems: ShellNavItem[] = [
    { to: "/achievements", label: text.achievements, icon: Trophy },
    { to: "/services/generic-program", label: text.plans, icon: ShoppingBag },
    { to: "/exercises", label: text.library, icon: BookOpen },
    ...(hasPremiumLevel2Academy ? [{ to: "/services/premium-level-2/academy", label: text.academy, icon: GraduationCap }] : []),
    { to: "/account/purchases", label: text.purchases, icon: ReceiptText },
    { to: "/account/settings", label: text.account, icon: Settings }
  ];
  const menuItems: UserMenuItem[] = [
    ...(user ? [{ to: profilePath(user.username), label: text.publicProfile, icon: UserRound }] : []),
    { to: "/account/settings", label: text.account, icon: Settings },
    { to: "/account/purchases", label: text.purchases, icon: ReceiptText },
    ...(hasPremiumLevel2Academy ? [{ to: "/services/premium-level-2/academy", label: text.academy, icon: GraduationCap }] : []),
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

  const name = user ? publicNameOf(user) : "";

  return (
    <MemberContext.Provider value={{ user, isTrainer, reload: () => setVersion((value) => value + 1) }}>
      {user?.needs_username ? <ChooseUsernameDialog onChosen={() => setVersion((value) => value + 1)} /> : null}
      <AppShell
        area="member"
        navLabel={text.nav}
        groups={[{ items }, { label: text.more, items: secondaryItems }]}
        // Phones keep four tabs (Recovery is in "More", which lists every section).
        tabs={items.filter((item) => item.to !== "/services/recovery")}
        moreLabel={text.more}
        closeLabel={text.close}
        user={
          user
            ? {
                name,
                email: user.email,
                // The @username, the unique reference, under the shown name
                // (the email is never shown as the account's identity).
                handle: user.username,
                avatarUrl: user.avatar_url ? publicFileUrl(user.avatar_url) : undefined,
                role: isTrainer ? undefined : text.member
              }
            : null
        }
        menu={{
          label: text.menu,
          items: menuItems,
          signOutLabel: text.signOut,
          signingOutLabel: text.signingOut,
          onSignOut: signOut
        }}
        action={{ to: "/services/next-workout", title: text.start, text: text.startText, icon: Play }}
        switchTo={isTrainer ? { to: "/trainer/dashboard", label: text.trainer, icon: Users } : undefined}
        backToSite={text.back}
        homePath="/profile"
        backLabel={text.goBack}
        parentOf={(pathname) => {
          const target = resolveBack(pathname);
          return { path: target.path, label: target.back[locale], viaHistory: target.viaHistory };
        }}
      >
        {children}
      </AppShell>
    </MemberContext.Provider>
  );
};
