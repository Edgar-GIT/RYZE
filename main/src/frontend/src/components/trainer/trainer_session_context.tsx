import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";

import { ApiError } from "@utils/http_client";
import { fetchTrainerAccess, type TrainerAccessStatus } from "@/services/trainer_api";

interface TrainerSessionState {
  access: TrainerAccessStatus | null;
  unauthenticated: boolean;
  loaded: boolean;
  refresh: () => void;
}

const TrainerSessionContext = createContext<TrainerSessionState | null>(null);

// TrainerSessionProvider resolves the authenticated user's trainer access
// state exactly once and shares it with every trainer surface. The backend
// response is the only source of truth: the provider never infers trainer
// status locally, and a 401 is remembered so route guards can redirect to the
// login page without guessing.
export const TrainerSessionProvider = ({ children }: { children: React.ReactNode }) => {
  const [access, setAccess] = useState<TrainerAccessStatus | null>(null);
  const [unauthenticated, setUnauthenticated] = useState(false);
  const [loaded, setLoaded] = useState(false);

  const load = useCallback(() => {
    let cancelled = false;

    fetchTrainerAccess()
      .then((result) => {
        if (!cancelled) {
          setAccess(result);
          setUnauthenticated(false);
          setLoaded(true);
        }
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setAccess(null);
          setUnauthenticated(error instanceof ApiError && error.status === 401);
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

  const value = useMemo<TrainerSessionState>(
    () => ({ access, unauthenticated, loaded, refresh: load }),
    [access, unauthenticated, loaded, load]
  );

  return <TrainerSessionContext.Provider value={value}>{children}</TrainerSessionContext.Provider>;
};

export const useTrainerSession = (): TrainerSessionState => {
  const ctx = useContext(TrainerSessionContext);
  if (!ctx) {
    throw new Error("useTrainerSession must be used within TrainerSessionProvider");
  }
  return ctx;
};
