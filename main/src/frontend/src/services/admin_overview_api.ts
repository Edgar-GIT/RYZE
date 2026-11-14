import { apiGet } from "@utils/http_client";

export interface OverviewDay {
  date: string;
  count: number;
  minor_units: number;
}

export interface OverviewShare {
  label: string;
  count: number;
  minor_units: number;
}

export interface OverviewProgram {
  id: string;
  name: string;
  family: string;
  count: number;
  minor_units: number;
}

export interface OverviewQueues {
  premium_level_2_awaiting_intake: number;
  premium_level_2_awaiting_trainer: number;
  premium_level_2_in_progress: number;
  premium_level_2_ready: number;
  support_open: number;
  feedback_pending: number;
  applications_pending?: number;
}

export interface BusinessOverview {
  generated_at: string;
  range_days: number;
  currency: string;
  revenue: {
    all_time_minor_units: number;
    all_time_sales: number;
    minor_units: number;
    previous_minor_units: number;
    sales: number;
    previous_sales: number;
    platform_minor_units: number;
    trainer_minor_units: number;
    by_family: OverviewShare[];
    daily: OverviewDay[];
  };
  purchases: { pending: number; failed: number; test: number };
  members: {
    total: number;
    new: number;
    signups_daily: OverviewDay[];
    active_7d: number;
    active_in_range: number;
  };
  training: {
    sessions_completed: number;
    sessions_daily: OverviewDay[];
    free_activations: number;
    active_entitlements: number;
  };
  trainers: { active: number; pending_applications: number };
  catalogue: { published_by_family: OverviewShare[]; platform_drafts: number };
  queues: OverviewQueues;
  top_selling: OverviewProgram[];
  top_viewed: OverviewProgram[];
}

export interface PlatformOverview {
  generated_at: string;
  range_days: number;
  members: {
    total: number;
    new: number;
    verified: number;
    disabled: number;
    signups_daily: OverviewDay[];
    active_7d: number;
  };
  training: { sessions_completed: number; sessions_daily: OverviewDay[] };
  trainers: { active: number; pending_applications: number };
  audit: { events_24h: number; denied_24h: number; failed_24h: number };
}

export const OVERVIEW_RANGES = [7, 30, 90] as const;

export const fetchBusinessOverview = (days: number) =>
  apiGet<BusinessOverview>(`/admin/overview/business?days=${days}`);

export const fetchPlatformOverview = (days: number) =>
  apiGet<PlatformOverview>(`/admin/overview/platform?days=${days}`);

export const fetchAdminQueues = () => apiGet<OverviewQueues>("/admin/overview/queues");

// Catalogue family labels used across the console.
export const FAMILY_LABEL: Record<string, string> = {
  generic: "Generic (free)",
  premium_level_1: "Premium Level 1",
  premium_level_2: "Premium Level 2",
  trainer: "Trainer plans"
};
