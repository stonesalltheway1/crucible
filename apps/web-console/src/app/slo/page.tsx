import { PageHeader, MetricStat } from "@/components/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { clampPercent } from "@/lib/utils";

const SLOS = [
  {
    name: "task_completion_within_estimate",
    description: "Tasks complete within the wall-clock and cost estimate shown in the plan.",
    objective: 90,
    actual: 93.4,
    window: "30d",
    status: "healthy",
  },
  {
    name: "promotion_canary_success",
    description: "Verified promotions pass canary without rollback.",
    objective: 99.5,
    actual: 99.7,
    window: "30d",
    status: "healthy",
  },
  {
    name: "verifier_decision_within_15min",
    description: "Tier 0 + Tier 1 verification completes within 15 minutes.",
    objective: 95,
    actual: 97.1,
    window: "30d",
    status: "healthy",
  },
  {
    name: "control_plane_availability",
    description: "Control plane API responsive (excluding planned maintenance).",
    objective: 99.9,
    actual: 99.94,
    window: "30d",
    status: "healthy",
  },
  {
    name: "attestation_publish_success",
    description: "All in-toto attestations successfully published to Rekor.",
    objective: 99.99,
    actual: 100.0,
    window: "30d",
    status: "healthy",
  },
] as const;

export default async function SloPage() {
  const violating = SLOS.filter((s) => s.actual < s.objective).length;
  const burning = SLOS.filter((s) => s.actual - s.objective < 0.2 && s.actual >= s.objective).length;

  return (
    <>
      <PageHeader
        title="SLO"
        description="The SLOs Crucible publishes to its customers. The same metrics that page our on-call."
      />

      <div className="mb-4 grid grid-cols-3 gap-3">
        <MetricStat label="Healthy" value={SLOS.length - violating - burning} tone="ok" />
        <MetricStat label="Burning" value={burning} tone={burning > 0 ? "warn" : "ok"} />
        <MetricStat label="Violated" value={violating} tone={violating > 0 ? "alert" : "ok"} />
      </div>

      <div className="space-y-3">
        {SLOS.map((s) => {
          const tone = s.actual >= s.objective ? "ok" : s.actual >= s.objective - 1 ? "warn" : "alert";
          const errBudgetUsedPct =
            s.objective === 100 ? 0 : clampPercent(((100 - s.actual) / (100 - s.objective)) * 100);
          return (
            <Card key={s.name}>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 font-mono">
                  {s.name}
                  <Badge tone={tone}>{s.actual.toFixed(2)}% / {s.objective}%</Badge>
                </CardTitle>
                <CardDescription>{s.description}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                <div>
                  <div className="mb-1 flex items-baseline justify-between">
                    <span className="font-mono text-[10px] uppercase text-muted-foreground">attainment</span>
                    <span className="font-mono text-xs tabular-nums">{s.actual.toFixed(2)}%</span>
                  </div>
                  <Progress value={s.actual} tone={tone} />
                </div>
                <div>
                  <div className="mb-1 flex items-baseline justify-between">
                    <span className="font-mono text-[10px] uppercase text-muted-foreground">error budget used</span>
                    <span className="font-mono text-xs tabular-nums">{errBudgetUsedPct.toFixed(0)}%</span>
                  </div>
                  <Progress value={errBudgetUsedPct} tone={errBudgetUsedPct < 70 ? "ok" : errBudgetUsedPct < 90 ? "warn" : "alert"} />
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </>
  );
}
