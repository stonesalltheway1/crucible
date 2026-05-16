import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { PlanSummary } from "@/components/plan-approval/plan-summary";

const plan = {
  description: "Add idempotency key to /webhooks/stripe/refund",
  estimated_cost_usd: 0.42,
  estimated_duration_min: 3,
  files_to_touch: ["api/webhooks/stripe.ts"],
  db_migrations: 0,
  external_effects: [],
  top_risks: [{ description: "double-refund regression", impact: "high" as const }],
  retry_budget_per_step: 3,
  wall_clock_budget_min: 15,
  hard_cap_usd: 2,
};

describe("PlanSummary", () => {
  it("renders the cost preview, file list, and risk callouts", () => {
    render(<PlanSummary plan={plan} />);
    expect(screen.getByText("$0.42")).toBeDefined();
    expect(screen.getByText("api/webhooks/stripe.ts")).toBeDefined();
    expect(screen.getByText(/double-refund regression/)).toBeDefined();
    expect(screen.getByText("high")).toBeDefined();
  });

  it("falls back gracefully when no external effects are present", () => {
    render(<PlanSummary plan={plan} />);
    expect(screen.getByText(/all calls hit the twin/)).toBeDefined();
  });
});
