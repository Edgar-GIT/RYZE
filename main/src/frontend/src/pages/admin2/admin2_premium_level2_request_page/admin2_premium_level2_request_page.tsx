import { useMemo } from "react";
import { useParams } from "react-router-dom";

import { PremiumLevel2Workspace, type WorkspaceAdapter } from "@/components/premium_level2_workspace/premium_level2_workspace";
import { fetchGenericPrograms, PROGRAM_PRODUCT_TYPE } from "@/services/admin2_api";
import {
  attachRyzePremiumLevel2TrainingProgram,
  deliverRyzePremiumLevel2Request,
  fetchAdminPremiumLevel2Catalog,
  fetchAdminPremiumLevel2Request,
  saveRyzePremiumLevel2Nutrition,
  uploadAdminMealImage
} from "@/services/premium_level_2_api";

const requestQuery = (requestId: string) => `request=${encodeURIComponent(requestId)}`;

// RYZE itself (ADMIN_2) prepares a Premium Level 2 request it took. Its plans
// to start from are the platform's free plans and Level 1 packages; the
// personal plan is a free platform draft linked to the request.
const RYZE_ADAPTER: WorkspaceAdapter = {
  backTo: "/admin2/premium-level-2",
  backLabel: "Premium L2 requests",
  eyebrow: "Premium Level 2 · prepared by RYZE",
  load: fetchAdminPremiumLevel2Request,
  loadCatalog: fetchAdminPremiumLevel2Catalog,
  loadPlans: async () => {
    const families = [PROGRAM_PRODUCT_TYPE.GENERIC, PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1];
    const lists = await Promise.all(families.map((family) => fetchGenericPrograms(family, { page: 1, limit: 100 })));
    return lists.flatMap((list, index) =>
      list.programs.map((program) => ({ id: program.id, name: program.name, status: program.status, family: families[index] }))
    );
  },
  attach: attachRyzePremiumLevel2TrainingProgram,
  saveNutrition: saveRyzePremiumLevel2Nutrition,
  deliver: deliverRyzePremiumLevel2Request,
  uploadMealImage: uploadAdminMealImage,
  newPlanLink: (requestId, base) =>
    `/admin2/plans/create?family=generic&${requestQuery(requestId)}${
      base ? `&base=${encodeURIComponent(base.id)}&baseFamily=${encodeURIComponent(base.family ?? "generic")}` : ""
    }`,
  canWork: (request) => Boolean(request.handled_by_ryze),
  editPlanLink: (requestId, programId) =>
    `/admin2/plans/create?edit=${encodeURIComponent(programId)}&family=generic&${requestQuery(requestId)}`
};

export default function Admin2PremiumLevel2RequestPage() {
  const { requestID } = useParams<{ requestID: string }>();
  const adapter = useMemo(() => RYZE_ADAPTER, []);
  return <PremiumLevel2Workspace requestId={requestID} adapter={adapter} />;
}
