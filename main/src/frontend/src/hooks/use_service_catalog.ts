import { useCallback, useEffect, useState } from "react";

import {
  fetchAllMarketplacePrograms,
  fetchAllPremiumLevel1Programs,
  type MarketplaceProgram
} from "@/services/marketplace_api";

export type ServiceCatalogStatus = "loading" | "error" | "ready";

export interface ServiceCatalogState {
  status: ServiceCatalogStatus;
  generic: MarketplaceProgram[];
  premium: MarketplaceProgram[];
  retry: () => void;
}

export const useServiceCatalog = (): ServiceCatalogState => {
  const [status, setStatus] = useState<ServiceCatalogStatus>("loading");
  const [generic, setGeneric] = useState<MarketplaceProgram[]>([]);
  const [premium, setPremium] = useState<MarketplaceProgram[]>([]);
  const [retryCount, setRetryCount] = useState(0);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setStatus("loading");
      const [genericResult, premiumResult] = await Promise.allSettled([
        fetchAllMarketplacePrograms(),
        fetchAllPremiumLevel1Programs()
      ]);
      if (cancelled) {
        return;
      }
      if (genericResult.status === "rejected" && premiumResult.status === "rejected") {
        setGeneric([]);
        setPremium([]);
        setStatus("error");
        return;
      }
      setGeneric(genericResult.status === "fulfilled" ? genericResult.value.programs : []);
      setPremium(premiumResult.status === "fulfilled" ? premiumResult.value.programs : []);
      setStatus("ready");
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, [retryCount]);

  const retry = useCallback(() => setRetryCount((count) => count + 1), []);

  return { status, generic, premium, retry };
};
