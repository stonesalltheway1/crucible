import Link from "next/link";
import { PageHeader, MetricStat } from "@/components/page-header";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { HashPill } from "@/components/hash-pill";
import { StatusBadge } from "@/components/status-badge";
import { PlanSummary } from "@/components/plan-approval/plan-summary";
import { StepTimeline } from "@/components/plan-approval/step-timeline";
import { ChainGraph } from "@/components/attestation/chain-graph";
import { formatDuration, formatRelative, formatUsd } from "@/lib/utils";
import { mockTask, mockAttestationChain } from "@/lib/mocks";
import { ExternalLink, FileSignature, Layers, ListChecks, ShieldCheck } from "lucide-react";

export default async function TaskDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const task = mockTask(id);
  const chain = mockAttestationChain(id);
  // Fake some completion so the surface shows the full state.
  task.status = "promotion_pending";
  task.cost_usd = 1.84;
  task.duration_seconds = 1320;
  task.steps = [
    {
      step_id: "step_01",
      name: "Plan + memory recall",
      status: "completed",
      cost_usd: 0.08,
      duration_seconds: 12,
      files_changed: [],
    },
    {
      step_id: "step_02",
      name: "Author handler + idempotency-key check",
      status: "completed",
      cost_usd: 0.42,
      duration_seconds: 47,
      files_changed: ["api/webhooks/stripe.ts"],
    },
    {
      step_id: "step_03",
      name: "Author repository module",
      status: "completed",
      cost_usd: 0.31,
      duration_seconds: 28,
      files_changed: ["db/idempotency_keys_repo.ts"],
    },
    {
      step_id: "step_04",
      name: "Write tests + run tier-0",
      status: "completed",
      cost_usd: 0.92,
      duration_seconds: 180,
      files_changed: ["api/webhooks/stripe.test.ts"],
    },
    {
      step_id: "step_05",
      name: "Cross-family verifier (tier-1 PBT)",
      status: "completed",
      cost_usd: 0.11,
      duration_seconds: 75,
      files_changed: [],
    },
  ];
  task.attestations = chain.nodes.map((n) => n.rekor_uuid);
  task.verifier = {
    verdict: "approved",
    rubric_score: 0.92,
    tier_results: {
      tier_0: { passed: true, mutation_score: 0.91 },
      tier_1: { passed: true, pbt_iterations: 10000, counterexamples: [] },
      tier_4: { passed: true, rebuild_hash: "be1cf9d3..." },
    },
    rejection_reasons: [],
    attestations: ["rekor:verifier-i9j0"],
    signed_by_oidc: "https://accounts.crucible.dev/agents/verifier-3",
    signed_at: new Date(Date.now() - 9 * 60_000).toISOString(),
  };

  return (
    <>
      <PageHeader
        title={task.description}
        description={
          <>
            <span className="font-mono">{task.repo}</span> · submitted by{" "}
            <span className="font-mono">{task.submitted_by}</span> · {formatRelative(task.submitted_at)}
          </>
        }
        actions={
          <>
            {task.status === "plan_pending_approval" && (
              <Button asChild>
                <Link href={`/tasks/${task.id}/approve`}>Review plan</Link>
              </Button>
            )}
            {task.status === "promotion_pending" && (
              <Button asChild>
                <Link href="/promotions">Open promotion</Link>
              </Button>
            )}
            {task.pr_url && (
              <Button asChild variant="outline">
                <a href={task.pr_url} target="_blank" rel="noopener noreferrer">
                  <ExternalLink className="h-3.5 w-3.5" /> Open PR
                </a>
              </Button>
            )}
          </>
        }
      />

      <div className="mb-4 grid grid-cols-4 gap-3">
        <MetricStat label="Status" value={<StatusBadge status={task.status} />} />
        <MetricStat
          label="Cost"
          value={formatUsd(task.cost_usd)}
          hint={`cap ${formatUsd(task.plan?.hard_cap_usd ?? 0)}`}
          tone={task.cost_usd <= (task.plan?.hard_cap_usd ?? 0) ? "ok" : "alert"}
        />
        <MetricStat label="Duration" value={formatDuration(task.duration_seconds)} />
        <MetricStat
          label="Verifier"
          value={task.verifier ? task.verifier.rubric_score.toFixed(2) : "—"}
          hint={task.verifier ? `rubric · ${task.verifier.verdict}` : "pending"}
          tone={task.verifier?.verdict === "approved" ? "ok" : task.verifier?.verdict === "rejected" ? "alert" : undefined}
        />
      </div>

      <Tabs defaultValue="plan">
        <TabsList>
          <TabsTrigger value="plan">
            <FileSignature className="mr-1 h-3.5 w-3.5" /> Plan
          </TabsTrigger>
          <TabsTrigger value="steps">
            <ListChecks className="mr-1 h-3.5 w-3.5" /> Steps
          </TabsTrigger>
          <TabsTrigger value="verifier">
            <ShieldCheck className="mr-1 h-3.5 w-3.5" /> Verifier
          </TabsTrigger>
          <TabsTrigger value="attestations">
            <Layers className="mr-1 h-3.5 w-3.5" /> Attestation chain
          </TabsTrigger>
        </TabsList>

        <TabsContent value="plan" className="pt-4">
          <Card>
            <CardContent className="p-4">{task.plan && <PlanSummary plan={task.plan} />}</CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="steps" className="pt-4">
          <Card>
            <CardContent className="p-4">
              <StepTimeline steps={task.steps ?? []} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="verifier" className="pt-4">
          {task.verifier ? (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  Verdict: <StatusBadge status={task.verifier.verdict} />
                </CardTitle>
                <CardDescription>
                  Signed by <span className="font-mono">{task.verifier.signed_by_oidc}</span> · {formatRelative(task.verifier.signed_at!)}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="grid grid-cols-3 gap-3">
                  <MetricStat label="Rubric" value={task.verifier.rubric_score.toFixed(2)} tone="ok" />
                  <MetricStat label="Mutation kill" value={`${Math.round((task.verifier.tier_results.tier_0?.mutation_score ?? 0) * 100)}%`} tone="ok" />
                  <MetricStat label="PBT iterations" value={String(task.verifier.tier_results.tier_1?.pbt_iterations ?? 0)} tone="ok" />
                </div>
                <div className="space-y-1">
                  <div className="font-mono text-[10px] uppercase tracking-wide text-muted-foreground">Tier results</div>
                  <div className="flex flex-wrap gap-1">
                    {Object.entries(task.verifier.tier_results).map(([k, v]) => (
                      <Badge key={k} tone={v.passed ? "ok" : "alert"}>{k}: {v.passed ? "pass" : "fail"}</Badge>
                    ))}
                  </div>
                </div>
              </CardContent>
            </Card>
          ) : (
            <Card>
              <CardContent className="p-6 text-sm text-muted-foreground">No verifier verdict yet.</CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="attestations" className="pt-4">
          <Card>
            <CardHeader>
              <CardTitle>Attestation chain</CardTitle>
              <CardDescription>
                The cryptographic trail from plan → file writes → verifier verdict → promotion approval. Every node is
                a Rekor-published in-toto attestation.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ChainGraph chain={chain} />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </>
  );
}
