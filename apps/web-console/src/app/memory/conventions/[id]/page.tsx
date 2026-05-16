import Link from "next/link";
import { notFound } from "next/navigation";
import { PageHeader, MetricStat } from "@/components/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { StatusBadge } from "@/components/status-badge";
import { formatRelative } from "@/lib/utils";
import { mockConventions } from "@/lib/mocks";
import { ConventionLifecycle } from "./_lifecycle";
import { ExternalLink, FileText, MessagesSquare, Siren } from "lucide-react";

const KIND_ICON: Record<string, React.ElementType> = {
  pr_comment: MessagesSquare,
  adr: FileText,
  agents_md: FileText,
  incident: Siren,
};

export default async function ConventionDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const c = mockConventions().find((x) => x.id === id);
  if (!c) notFound();

  return (
    <>
      <PageHeader
        title={c.rule_nl}
        description={
          <>
            <Badge tone="mute">{c.category}</Badge>{" "}
            <span className="font-mono">{c.scope.repo}</span>
            {c.scope.file_glob && <span className="font-mono"> · {c.scope.file_glob}</span>}
          </>
        }
        actions={<StatusBadge status={c.status} />}
      />

      <div className="mb-4 grid grid-cols-4 gap-3">
        <MetricStat
          label="Confidence"
          value={c.confidence.toFixed(2)}
          hint={`A-MAC, Platt-scaled`}
          tone={c.confidence > 0.8 ? "ok" : c.confidence > 0.6 ? "warn" : "alert"}
        />
        <MetricStat label="+ examples 30d" value={c.positive_examples_30d} tone="ok" />
        <MetricStat
          label="− examples 30d"
          value={c.negative_examples_30d}
          tone={c.negative_examples_30d > 0 ? "warn" : "ok"}
        />
        <MetricStat
          label="Last violation"
          value={c.last_violated_at ? formatRelative(c.last_violated_at) : "—"}
          tone={c.status === "drifting" ? "warn" : undefined}
        />
      </div>

      <div className="grid grid-cols-[1fr_320px] gap-4">
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Source evidence</CardTitle>
              <CardDescription>
                The conversations the distiller extracted this rule from. The LLM judge sees only the rule text, never
                the source — this list is for human review.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {c.source_evidence.length === 0 ? (
                <div className="text-xs text-muted-foreground">none recorded</div>
              ) : (
                <ul className="space-y-2">
                  {c.source_evidence.map((e, i) => {
                    const Icon = KIND_ICON[e.kind] ?? FileText;
                    return (
                      <li key={i} className="flex items-start gap-2 border border-ink-200 p-2 dark:border-ink-800">
                        <Icon className="mt-0.5 h-3.5 w-3.5 text-muted-foreground" />
                        <div className="min-w-0 flex-1 text-sm">
                          <div className="flex items-center gap-2">
                            <Badge tone="mute">{e.kind.replace("_", " ")}</Badge>
                            {e.ref && <span className="font-mono text-xs">{e.ref}</span>}
                            {e.url && (
                              <a className="inline-flex items-center gap-1 text-xs underline-offset-2 hover:underline" href={e.url} target="_blank" rel="noopener noreferrer">
                                open <ExternalLink className="h-3 w-3" />
                              </a>
                            )}
                          </div>
                          {e.excerpt && <div className="mt-1 text-xs italic text-muted-foreground">“{e.excerpt}”</div>}
                        </div>
                      </li>
                    );
                  })}
                </ul>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Compliance signal</CardTitle>
              <CardDescription>
                When the verifier checks this rule, violations contribute to the rubric's `trust_signal_alignment` slot.
                Phase 5 caps this at severity=warn — Phase 7 wires the `rule_machine` matcher for severity=error.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-2">
                <Progress value={c.confidence * 100} tone={c.confidence > 0.8 ? "ok" : "warn"} />
                <span className="font-mono text-xs tabular-nums">{(c.confidence * 100).toFixed(0)}%</span>
              </div>
              <div className="mt-2 text-xs text-muted-foreground">
                +{c.positive_examples_30d} positive · −{c.negative_examples_30d} negative · ratio{" "}
                {(c.positive_examples_30d / Math.max(1, c.negative_examples_30d)).toFixed(1)} (drift threshold 1.5)
              </div>
            </CardContent>
          </Card>

          {c.superseded_by && (
            <Card>
              <CardHeader>
                <CardTitle>Superseded</CardTitle>
                <CardDescription>This rule was replaced by a newer one. History is kept for audit.</CardDescription>
              </CardHeader>
              <CardContent>
                <Link
                  className="font-mono text-xs underline-offset-2 hover:underline"
                  href={`/memory/conventions/${c.superseded_by}`}
                >
                  → {c.superseded_by}
                </Link>
              </CardContent>
            </Card>
          )}
        </div>

        <aside className="space-y-3">
          <ConventionLifecycle conventionId={c.id} status={c.status} />

          <Card>
            <CardHeader>
              <CardTitle>Identity</CardTitle>
            </CardHeader>
            <CardContent className="space-y-1.5 text-xs">
              <div className="flex items-baseline justify-between gap-2">
                <span className="font-mono text-[10px] uppercase text-muted-foreground">id</span>
                <span className="font-mono text-xs">{c.id}</span>
              </div>
              <div className="flex items-baseline justify-between gap-2">
                <span className="font-mono text-[10px] uppercase text-muted-foreground">layer</span>
                <Badge
                  tone={
                    c.source_layer === "customer" ? "ok" : c.source_layer === "tenant_distilled" ? "info" : "mute"
                  }
                >
                  {c.source_layer.replace("_", " ")}
                </Badge>
              </div>
            </CardContent>
          </Card>
        </aside>
      </div>
    </>
  );
}
