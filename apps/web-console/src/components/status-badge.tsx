import { Badge } from "@/components/ui/badge";

type Status =
  | "submitted"
  | "planning"
  | "plan_pending_approval"
  | "plan_approved"
  | "executing"
  | "verifying"
  | "verified"
  | "verification_failed"
  | "promotion_pending"
  | "promoted"
  | "completed"
  | "failed"
  | "cancelled"
  | "pending_approval"
  | "approved"
  | "rejected"
  | "deploying"
  | "canary_dwell"
  | "landed"
  | "rolled_back"
  | "candidate"
  | "active"
  | "drifting"
  | "superseded";

const TONE: Record<Status, "ok" | "warn" | "alert" | "info" | "mute"> = {
  submitted: "info",
  planning: "info",
  plan_pending_approval: "warn",
  plan_approved: "info",
  executing: "info",
  verifying: "info",
  verified: "ok",
  verification_failed: "alert",
  promotion_pending: "warn",
  promoted: "ok",
  completed: "ok",
  failed: "alert",
  cancelled: "mute",
  pending_approval: "warn",
  approved: "info",
  rejected: "alert",
  deploying: "info",
  canary_dwell: "info",
  landed: "ok",
  rolled_back: "alert",
  candidate: "info",
  active: "ok",
  drifting: "warn",
  superseded: "mute",
};

const LABEL: Partial<Record<Status, string>> = {
  plan_pending_approval: "plan: pending",
  plan_approved: "plan: approved",
  verification_failed: "verify: failed",
  promotion_pending: "promote: pending",
  pending_approval: "pending approval",
  canary_dwell: "canary",
  rolled_back: "rolled back",
};

export function StatusBadge({ status }: { status: Status | string }) {
  const s = status as Status;
  const tone = TONE[s] ?? "mute";
  const label = LABEL[s] ?? status.replace(/_/g, " ");
  return <Badge tone={tone}>{label}</Badge>;
}
