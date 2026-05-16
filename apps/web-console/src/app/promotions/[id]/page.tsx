import Link from "next/link";
import { PageHeader, MetricStat } from "@/components/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/status-badge";
import { HashPill } from "@/components/hash-pill";
import { formatDuration, formatRelative } from "@/lib/utils";
import { mockPromotion } from "@/lib/mocks";
import { ApprovalActions } from "./_approval-actions";
import { CanaryStrip } from "@/components/promotion/canary-strip";
import { CheckCheck, FileEdit, ShieldAlert, ShieldCheck } from "lucide-react";

export default async function PromotionDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const p = mockPromotion(id);

  return (
    <>
      <PageHeader
        title={`Promotion ${p.id}`}
        description={
          <>
            task <span className="font-mono">{p.task_id}</span> · submitted {formatRelative(p.submitted_at)}
          </>
        }
        actions={<StatusBadge status={p.status} />}
      />

      <div className="grid grid-cols-[1fr_360px] gap-4">
        <div className="space-y-4">
          {p.canary && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <CheckCheck className="h-3.5 w-3.5" /> Canary progression
                </CardTitle>
                <CardDescription>
                  Adapter <span className="font-mono">{p.canary.adapter}</span> · auto-rollback fires within one
                  SLO-check cycle of regression.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <CanaryStrip steps={p.canary.steps} current={p.canary.current_step} />
              </CardContent>
            </Card>
          )}

          {p.decision && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <ShieldCheck className="h-3.5 w-3.5" /> Policy decision
                </CardTitle>
                <CardDescription>
                  Trace path <span className="font-mono">{p.decision.trace?.path}</span> · policy hash{" "}
                  {p.decision.trace?.policy_hash && <HashPill value={p.decision.trace.policy_hash} />}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex flex-wrap gap-1">
                  <Badge tone={p.decision.allow ? "ok" : "alert"}>{p.decision.allow ? "allow" : "deny"}</Badge>
                  {p.decision.needs_human && <Badge tone="warn">needs human</Badge>}
                  {p.decision.auto_approve && <Badge tone="ok">auto-approve</Badge>}
                </div>
                {p.decision.approver_groups && p.decision.approver_groups.length > 0 && (
                  <div>
                    <div className="font-mono text-[10px] uppercase text-muted-foreground">Approver groups</div>
                    <div className="mt-1 flex flex-wrap gap-1">
                      {p.decision.approver_groups.map((g) => (
                        <Badge key={g} tone="info">{g}</Badge>
                      ))}
                      <Badge tone="mute">require {p.decision.require_n_approvers}</Badge>
                    </div>
                  </div>
                )}
                {p.decision.reasons && (
                  <div>
                    <div className="font-mono text-[10px] uppercase text-muted-foreground">Rules fired</div>
                    <ul className="mt-1 list-disc pl-5 text-xs">
                      {p.decision.reasons.map((r) => (
                        <li key={r}>{r}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {p.bundle && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <FileEdit className="h-3.5 w-3.5" /> Bundle
                </CardTitle>
                <CardDescription>The diff that lands if approved. The hash is what every signature binds to.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex items-center gap-2">
                  <span className="font-mono text-[10px] uppercase text-muted-foreground">diff_hash</span>
                  <HashPill value={p.bundle.diff_hash} />
                </div>
                <ul className="space-y-1">
                  {p.bundle.files_changed.map((f) => (
                    <li key={f.path} className="flex items-center gap-2 font-mono text-xs">
                      <Badge tone={f.action === "create" ? "ok" : f.action === "delete" ? "alert" : "info"}>{f.action}</Badge>
                      <span>{f.path}</span>
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
          )}

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ShieldAlert className="h-3.5 w-3.5" /> Approvals
              </CardTitle>
              <CardDescription>
                Self-approval is rejected at every layer (relay, Rego, gate, Slack). Sigstore keyless OIDC binds each signature to a human.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {p.approvals.length === 0 ? (
                <div className="text-xs text-muted-foreground">No approvals yet.</div>
              ) : (
                <ul className="space-y-2">
                  {p.approvals.map((a) => (
                    <li key={a.attestation} className="flex flex-wrap items-center justify-between gap-2 border border-ink-200 bg-ink-50 p-2 dark:border-ink-800 dark:bg-ink-900">
                      <div>
                        <div className="text-sm">{a.approver_oidc_subject}</div>
                        <div className="font-mono text-[10px] uppercase text-muted-foreground">
                          {a.group} · {formatRelative(a.approved_at)}
                        </div>
                      </div>
                      <HashPill value={a.attestation} href={`/attestations/${encodeURIComponent(a.attestation)}`} />
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>
        </div>

        <aside className="space-y-3">
          <ApprovalActions promotionId={p.id} status={p.status} bundleHash={p.bundle?.diff_hash ?? ""} />

          <Card>
            <CardHeader>
              <CardTitle>Identity</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-xs">
              <div className="flex items-baseline justify-between gap-2">
                <span className="font-mono text-[10px] uppercase text-muted-foreground">task</span>
                <Link className="font-mono text-xs underline-offset-2 hover:underline" href={`/tasks/${p.task_id}`}>
                  {p.task_id}
                </Link>
              </div>
              {p.bundle && (
                <div className="flex items-baseline justify-between gap-2">
                  <span className="font-mono text-[10px] uppercase text-muted-foreground">agent OIDC</span>
                  <span className="font-mono text-xs">{p.bundle.agent_oidc_subject}</span>
                </div>
              )}
              {p.bundle && (
                <div className="flex items-baseline justify-between gap-2">
                  <span className="font-mono text-[10px] uppercase text-muted-foreground">blast radius</span>
                  <span className="font-mono text-xs">
                    {p.bundle.blast_radius.reversibility} · {p.bundle.blast_radius.impact_score.toFixed(2)}
                  </span>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Quick links</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-xs">
              <Link className="block underline-offset-2 hover:underline" href={`/tasks/${p.task_id}`}>
                → Task detail + attestation chain
              </Link>
              <Link className="block underline-offset-2 hover:underline" href="/attestations">
                → Search attestations
              </Link>
            </CardContent>
          </Card>
        </aside>
      </div>
    </>
  );
}
