import Link from "next/link";
import { PageHeader } from "@/components/page-header";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { StatusBadge } from "@/components/status-badge";
import { HashPill } from "@/components/hash-pill";
import { formatDuration, formatRelative, formatUsd } from "@/lib/utils";
import { mockTaskList } from "@/lib/mocks";
import { Plus, Search } from "lucide-react";
import { TaskFilters } from "./_filters";

export default async function TasksPage() {
  const tasks = mockTaskList();
  return (
    <>
      <PageHeader
        title="Tasks"
        description="Every task ever run on this tenant. Cost, duration, verifier verdict, and the attestation chain are linked from the detail page."
        actions={
          <>
            <Button variant="outline" asChild>
              <Link href="/tasks/new">
                <Plus className="h-3.5 w-3.5" /> New task
              </Link>
            </Button>
          </>
        }
      />
      <TaskFilters />
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[36%]">Description</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Submitted</TableHead>
                <TableHead>By</TableHead>
                <TableHead className="text-right">Cost</TableHead>
                <TableHead className="text-right">Duration</TableHead>
                <TableHead>Task ID</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tasks.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>
                    <Link href={`/tasks/${t.id}`} className="font-medium underline-offset-2 hover:underline">
                      {t.description}
                    </Link>
                    <div className="font-mono text-[10px] text-muted-foreground">{t.repo}</div>
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={t.status} />
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">{formatRelative(t.submitted_at)}</TableCell>
                  <TableCell className="text-xs">{t.submitted_by}</TableCell>
                  <TableCell className="text-right tabular-nums">
                    {t.cost_usd > 0 ? formatUsd(t.cost_usd) : "—"}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {t.duration_seconds > 0 ? formatDuration(t.duration_seconds) : "—"}
                  </TableCell>
                  <TableCell>
                    <HashPill value={t.id} href={`/tasks/${t.id}`} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </>
  );
}
