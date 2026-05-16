import Link from "next/link";
import { PageHeader, MetricStat } from "@/components/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { StatusBadge } from "@/components/status-badge";
import { HashPill } from "@/components/hash-pill";
import { formatRelative } from "@/lib/utils";
import { mockPromotions } from "@/lib/mocks";

export default async function PromotionsPage() {
  const all = mockPromotions();
  const pending = all.filter((p) => p.status === "pending_approval" || p.status === "canary_dwell");
  const recent = all.filter((p) => p.status !== "pending_approval" && p.status !== "canary_dwell");

  return (
    <>
      <PageHeader
        title="Promotions"
        description="Verified bundles waiting on a human, plus the recent history. Every promotion's KMS lease is time-boxed, action-scoped, and single-use."
      />

      <div className="mb-4 grid grid-cols-4 gap-3">
        <MetricStat label="Pending approval" value={all.filter((p) => p.status === "pending_approval").length} tone="warn" />
        <MetricStat label="Canary dwelling" value={all.filter((p) => p.status === "canary_dwell").length} />
        <MetricStat label="Landed (24h)" value={all.filter((p) => p.status === "landed").length} tone="ok" />
        <MetricStat label="Rolled back (24h)" value={all.filter((p) => p.status === "rolled_back").length} tone="alert" />
      </div>

      <Tabs defaultValue="inbox">
        <TabsList>
          <TabsTrigger value="inbox">Approval inbox ({pending.length})</TabsTrigger>
          <TabsTrigger value="history">Recent history</TabsTrigger>
        </TabsList>

        <TabsContent value="inbox" className="pt-4">
          {pending.length === 0 ? (
            <Card>
              <CardContent className="p-6 text-sm text-muted-foreground">Nothing waiting on you.</CardContent>
            </Card>
          ) : (
            <div className="space-y-3">
              {pending.map((p) => (
                <Card key={p.id}>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Link href={`/promotions/${p.id}`} className="underline-offset-2 hover:underline">
                        Promotion {p.id}
                      </Link>
                      <StatusBadge status={p.status} />
                    </CardTitle>
                    <CardDescription>
                      task <span className="font-mono">{p.task_id}</span> · submitted {formatRelative(p.submitted_at)}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    {p.decision && (
                      <div className="border border-ink-200 bg-ink-50 p-2 dark:border-ink-800 dark:bg-ink-900">
                        <div className="font-mono text-[10px] uppercase tracking-wide text-muted-foreground">
                          Rego decision · trace {p.decision.trace?.path}
                        </div>
                        <ul className="mt-1 list-disc pl-5 text-xs">
                          {(p.decision.reasons ?? []).map((r) => (
                            <li key={r}>{r}</li>
                          ))}
                        </ul>
                      </div>
                    )}
                    <div className="flex items-center justify-between gap-2">
                      <div className="text-xs text-muted-foreground">
                        Approvers: {(p.decision?.approver_groups ?? []).join(", ")} · require{" "}
                        {p.decision?.require_n_approvers}
                      </div>
                      <div className="flex gap-2">
                        <Button asChild variant="outline" size="sm">
                          <Link href={`/promotions/${p.id}`}>Review</Link>
                        </Button>
                        <Button asChild size="sm">
                          <Link href={`/promotions/${p.id}#approve`}>Approve</Link>
                        </Button>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>

        <TabsContent value="history" className="pt-4">
          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Promotion</TableHead>
                    <TableHead>Task</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Submitted</TableHead>
                    <TableHead>Diff hash</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {recent.map((p) => (
                    <TableRow key={p.id}>
                      <TableCell>
                        <Link className="underline-offset-2 hover:underline" href={`/promotions/${p.id}`}>
                          {p.id}
                        </Link>
                      </TableCell>
                      <TableCell>
                        <HashPill value={p.task_id} href={`/tasks/${p.task_id}`} />
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={p.status} />
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">{formatRelative(p.submitted_at)}</TableCell>
                      <TableCell>{p.bundle?.diff_hash && <HashPill value={p.bundle.diff_hash} />}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </>
  );
}
