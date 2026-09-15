import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { AdminIdentity, fetchAdminIdentity } from "@/services/admin_api";

interface Admin2SessionState {
  identity: AdminIdentity | null;
  loaded: boolean;
  refresh: () => void;
}

const Admin2SessionContext = createContext<Admin2SessionState | null>(null);

export const Admin2SessionProvider = ({ children }: { children: React.ReactNode }) => {
  const [identity, setIdentity] = useState<AdminIdentity | null>(null);
  const [loaded, setLoaded] = useState(false);

  const load = useCallback(() => {
    let cancelled = false;

    fetchAdminIdentity()
      .then((res) => {
        if (!cancelled) {
          setIdentity(res ?? null);
          setLoaded(true);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setIdentity(null);
          setLoaded(true);
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    return load();
  }, [load]);

  const value = useMemo<Admin2SessionState>(
    () => ({ identity, loaded, refresh: load }),
    [identity, loaded, load]
  );

  return <Admin2SessionContext.Provider value={value}>{children}</Admin2SessionContext.Provider>;
};

export const useAdmin2Session = (): Admin2SessionState => {
  const ctx = useContext(Admin2SessionContext);
  if (!ctx) {
    throw new Error("useAdmin2Session must be used within Admin2SessionProvider");
  }
  return ctx;
};