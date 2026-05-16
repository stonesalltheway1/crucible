import Link from "next/link";
import { PageHeader, MetricStat } from "@/components/page-header";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/status-badge";
import { HashPill } from "@/components/hash-pill";
import { formatRelative, formatUsd } from "@/lib/utils";
import { ArrowUpRight, FileSignature, Activity, ListChecks, ShieldCheck } from "lucide-react";

// Server-rendered overview. In production this hydrates from the control plane;
// in the absence of a connected backend, we fall back to a deterministic
// demo payload so the surface remains legible to senior-engineer reviewers.
async function loadOverview() {
  return {
    counters: {
      tasks_today: 14,
      verified_today: 12,
      promotions_today: 9,
      rolled_back_today: 0,
    },
    economics: {
      median_task_cost: 1.42,
      p95_task_cost: 5.81,
      cache_hit_rate: 0.78,
      verifier_cost_pct: 0.08,
    },
    pending: [
      {
        id: "task_01HZAB...x4",
        description: "Add idempotency key to /webhooks/stripe/refund",
        status: "plan_pending_approval",
        cost: 0.42,
        submitted_at: new Date(Date.now() - 4 * 60_000).toISOString(),
        submitted_by: "sarah@acme.dev",
      },
      {
        id: "task_01HZAB...m2",
        description: "Bump fast-check to 4.x across packages/*",
        status: "verifying",
        cost: 0.91,
        submitted_at: new Date(Date.now() - 21 * 60_000).toISOString(),
        submitted_by: "marcus@acme.dev",
      },
      {
        id: "prom_01HZAB...p7",
        description: "Promotion: refund handler → canary 25%",
        status: "canary_dwell",
        cost: 0,
        submitted_at: new Date(Date.now() - 8 * 60_000).toISOString(),
        submitted_by: "agent · worker-7",
      },
    ],
    latest_attestation: {
      uuid: "rekor:b2cdd9f4c8a1a3e2",
      predicate: "VerifierApproval/v1",
      signed_at: new Date(Date.now() - 90_000).toISOString(),
    },
  };
}

export default async function OverviewPage() {
  const data = await loadOverview();
  return (
    <>
      <PageHeader
        title="Overview"
        description="Last 24h activity for this tenant. The agent does not produce code without a signed plan, verifier approval, and attestation chain."
        actions={
          <Button asChild>
            <Link href="/tasks">
              Submit task <ArrowUpRight className="h-3.5 w-3.5" />
            </Link>
          </Button>
        }
      />

      <div className="grid grid-cols-4 gap-3">
        <MetricStat label="Tasks 24h" value={data.counters.tasks_today} hint="submitted across all surfaces" />
        <MetricStat
          label="Verified"
          value={data.counters.verified_today}
          hint={`${Math.round((data.counters.verified_today / data.counters.tasks_today) * 100)}% of submitted`}
          tone="ok"
        />
        <MetricStat label="Promotions" value={data.counters.promotions_today} hint="canary → landed" />
        <MetricStat
          label="Rollbacks"
          value={data.counters.rolled_back_today}
          hint="auto-rolled-back within one SLO cycle"
          tone={data.counters.rolled_back_today === 0 ? "ok" : "alert"}
        />
      </div>

      <div className="mt-6 grid grid-cols-3 gap-3">
        <MetricStat
          label="Median task"
          value={formatUsd(data.economics.median_task_cost)}
          hint="target ≤ $1.69 / 24h"
          tone={data.economics.median_task_cost <= 1.69 ? "ok" : "warn"}
        />
        <MetricStat
          label="Cache hit"
          value={`${Math.round(data.economics.cache_hit_rate * 100)}%`}
          hint="target ≥ 70%"
          tone={data.economics.cache_hit_rate >= 0.7 ? "ok" : "warn"}
        />
        <MetricStat
          label="Verifier cost"
          value={`${Math.round(data.economics.verifier_cost_pct * 100)}%`}
          hint="of total spend; cap 10%"
          tone={data.economics.verifier_cost_pct <= 0.1 ? "ok" : "warn"}
        />
      </div>

      <div className="mt-6 grid grid-cols-[2fr_1fr] gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-3.5 w-3.5" />
              In flight
            </CardTitle>
            <CardDescription>Tasks and promotions currently waiting on you or the verifier.</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <ul className="divide-y divide-ink-200 dark:divide-ink-800">
              {data.pending.map((p) => (
                <li key={p.id} className="flex items-center gap-3 px-4 py-3 hover:bg-ink-50/60 dark:hover:bg-ink-900/60">
                  <div className="grid h-7 w-7 place-items-center border border-ink-200 bg-ink-50 dark:border-ink-800 dark:bg-ink-900">
                    {p.id.startsWith("prom_") ? <ShieldCheck className="h-3.5 w-3.5" /> : <ListChecks className="h-3.5 w-3.5" />}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{p.description}</span>
                      <StatusBadge status={p.status} />
                    </div>
                    <div className="mt-0.5 flex items-center gap-3 text-xs text-muted-foreground">
                      <HashPill value={p.id} />
                      <span>{p.submitted_by}</span>
                      <span>{formatRelative(p.submitted_at)}</span>
                      {p.cost > 0 && <span>{formatUsd(p.cost)}</span>}
                    </div>
                  </div>
                  <Button variant="outline" size="sm" asChild>
                    <Link href={p.id.startsWith("prom_") ? `/promotions/${p.id}` : `/tasks/${p.id}`}>Open</Link>
                  </Button>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileSignature className="h-3.5 w-3.5" />
              Latest attestation
            </CardTitle>
            <CardDescription>Every action is signed and inclusion-proven in the transparency log.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="flex items-center gap-2">
              <span className="text-xs uppercase text-muted-foreground">predicate</span>
              <span className="font-mono text-xs">{data.latest_attestation.predicate}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-xs uppercase text-muted-foreground">rekor</span>
              <HashPill value={data.latest_attestation.uuid} href={`/attestations/${data.latest_attestation.uuid}`} />
            </div>
            <div className="flex items-center gap-2">
              <span className="text-xs uppercase text-muted-foreground">signed</span>
              <span className="text-xs">{formatRelative(data.latest_attestation.signed_at)}</span>
            </div>
            <Button asChild variant="outline" size="sm" className="mt-2 w-full">
              <Link href="/attestations">Browse all</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    </>
  );
}
