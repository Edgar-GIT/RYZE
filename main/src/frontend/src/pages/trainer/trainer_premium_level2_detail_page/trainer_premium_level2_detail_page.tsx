import { useMemo } from "react";
import { useParams } from "react-router-dom";

import { PremiumLevel2Workspace, type WorkspaceAdapter } from "@/components/premium_level2_workspace/premium_level2_workspace";
import {
  attachPremiumLevel2TrainingProgram,
  deliverPremiumLevel2Request,
  fetchTrainerPremiumLevel2Catalog,
  fetchTrainerPremiumLevel2Request,
  savePremiumLevel2Nutrition,
  uploadTrainerMealImage
} from "@/services/premium_level_2_api";
import { fetchTrainerPrograms } from "@/services/trainer_api";

const planPath = (requestId: string) => `request=${encodeURIComponent(requestId)}`;

// The trainer who accepted a Premium Level 2 request prepares it here.
const TRAINER_ADAPTER: WorkspaceAdapter = {
  backTo: "/trainer/premium-level-2",
  backLabel: "Client services",
  eyebrow: "Premium Level 2",
  load: fetchTrainerPremiumLevel2Request,
  loadCatalog: fetchTrainerPremiumLevel2Catalog,
  loadPlans: async () => {
    const result = await fetchTrainerPrograms(1, 100);
    return result.programs.map((program) => ({ id: program.id, name: program.name, status: program.status }));
  },
  attach: attachPremiumLevel2TrainingProgram,
  saveNutrition: savePremiumLevel2Nutrition,
  deliver: deliverPremiumLevel2Request,
  uploadMealImage: uploadTrainerMealImage,
  newPlanLink: (requestId, base) =>
    `/trainer/programs/new?${planPath(requestId)}${base ? `&base=${encodeURIComponent(base.id)}` : ""}`,
  editPlanLink: (requestId, programId) => `/trainer/programs/${encodeURIComponent(programId)}/edit?${planPath(requestId)}`
};

export const TrainerPremiumLevel2DetailPage = () => {
  const { requestID } = useParams<{ requestID: string }>();
  const adapter = useMemo(() => TRAINER_ADAPTER, []);
  return <PremiumLevel2Workspace requestId={requestID} adapter={adapter} />;
};
