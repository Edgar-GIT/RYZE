import { useCallback } from "react";
import { useHistory, useLocation } from "react-router-dom";

import { useI18n } from "@/i18n/i18n";
import { resolveBack } from "@/services/navigation_history";

// useBackTarget is the in-page "back" of a public page, deciding like the
// navbar button (`resolveBack`): to the page the person came from when that
// was another area or the page above, otherwise up to the parent page.
export const useBackTarget = () => {
  const { locale } = useI18n();
  const history = useHistory();
  const { pathname } = useLocation();
  const target = resolveBack(pathname);
  const goBack = useCallback(() => history.goBack(), [history]);
  return {
    label: target.back[locale],
    // A link when going up; a history step when returning to the previous page.
    to: target.viaHistory ? undefined : target.path,
    onClick: target.viaHistory ? goBack : undefined
  };
};
