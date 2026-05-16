import Link from "next/link";
import { PageHeader, MetricStat } from "@/components/page-header";
import { Card, CardContent } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/status-badge";
import { Progress } from "@/components/ui/progress";
import { formatRelative } from "@/lib/utils";
import { mockConventions } from "@/lib/mocks";
import { MemoryFilters } from "./_filters";

export default async function MemoryPage() {
  const all = mockConventions();
  const active = all.filter((c) => c.status === "active");
  const drifting = all.filter((c) => c.status === "drifting");
  const candidate = all.filter((c) => c.status === "candidate");
  const superseded = all.filter((c) => c.status === "superseded");

  return (
    <>
      <PageHeader
        title="Memory"
        description="Per-tenant procedural memory. Conventions distilled from PR review comments, ADRs, incidents, and lint configs. The customer's AGENTS.md / CLAUDE.md / .cursorrules always wins."
      />

      <div className="mb-4 grid grid-cols-4 gap-3">
        <MetricStat label="Active" value={active.length} tone="ok" />
        <MetricStat label="Drifting" value={drifting.length} tone={drifting.length > 0 ? "warn" : "ok"} />
        <MetricStat label="Candidates" value={candidate.length} />
        <MetricStat label="Superseded" value={superseded.length} />
      </div>

      <Tabs defaultValue="active">
        <TabsList>
          <TabsTrigger value="active">Active ({active.length})</TabsTrigger>
          <TabsTrigger value="drifting">Drifting ({drifting.length})</TabsTrigger>
          <TabsTrigger value="candidate">Candidates ({candidate.length})</TabsTrigger>
          <TabsTrigger value="superseded">Superseded ({superseded.length})</TabsTrigger>
        </TabsList>

        {[
          { v: "active", rows: active },
          { v: "drifting", rows: drifting },
          { v: "candidate", rows: candidate },
          { v: "superseded", rows: superseded },
        ].map(({ v, rows }) => (
          <TabsContent key={v} value={v} className="pt-4 space-y-3">
            <MemoryFilters />
            <Card>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-[44%]">Rule</TableHead>
                      <TableHead>Category</TableHead>
                      <TableHead>Layer</TableHead>
                      <TableHead className="text-right">Confidence</TableHead>
                      <TableHead className="text-right">+/− 30d</TableHead>
                      <TableHead>Last violation</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {rows.length === 0 && (
                      <TableRow>
                        <TableCell colSpan={6} className="py-6 text-center text-xs text-muted-foreground">
                          nothing here
                        </TableCell>
                      </TableRow>
                    )}
                    {rows.map((c) => (
                      <TableRow key={c.id}>
                        <TableCell>
                          <Link
                            href={`/memory/conventions/${c.id}`}
                            className="text-sm font-medium underline-offset-2 hover:underline"
                          >
                            {c.rule_nl}
                          </Link>
                          <div className="font-mono text-[10px] text-muted-foreground">
                            {c.scope.repo}
                            {c.scope.file_glob ? ` · ${c.scope.file_glob}` : ""}
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge tone="mute">{c.category}</Badge>
                        </TableCell>
                        <TableCell>
                          <Badge
                            tone={
                              c.source_layer === "customer"
                                ? "ok"
                                : c.source_layer === "tenant_distilled"
                                  ? "info"
                                  : "mute"
                            }
                          >
                            {c.source_layer.replace("_", " ")}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-2">
                            <Progress value={c.confidence * 100} className="h-1.5 w-16" tone={c.confidence > 0.8 ? "ok" : c.confidence > 0.6 ? "warn" : "alert"} />
                            <span className="font-mono text-xs tabular-nums">{c.confidence.toFixed(2)}</span>
                          </div>
                        </TableCell>
                        <TableCell className="text-right font-mono text-xs tabular-nums">
                          <span className="text-accent-ok">+{c.positive_examples_30d}</span> /{" "}
                          <span className="text-accent-alert">−{c.negative_examples_30d}</span>
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {c.last_violated_at ? formatRelative(c.last_violated_at) : "—"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </TabsContent>
        ))}
      </Tabs>
    </>
  );
}
