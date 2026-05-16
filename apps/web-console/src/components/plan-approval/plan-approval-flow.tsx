"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Textarea } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { StatusBadge } from "@/components/status-badge";
import { HashPill } from "@/components/hash-pill";
import { PlanSummary } from "./plan-summary";
import { BudgetViewer, type BudgetConfig } from "./budget-viewer";
import { StepTimeline } from "./step-timeline";
import { useSse } from "@/lib/sse";
import { useToast } from "@/components/ui/toaster";
import { api, type Plan, type Step, type Task } from "@/lib/api";
import { useTenant } from "@/lib/tenant-context";
import { CircleStop, FileSignature, MessagesSquare } from "lucide-react";

// The plan-approval surface. The brief calls this the customer-trust signature
// surface — every detail (cost preview, hard cap, retry budget, live stream,
// interrupt) is the visible expression of the trust narrative.

const editsSchema = z.object({ amendment: z.string().max(2000).optional() });

export function PlanApprovalFlow({ task }: { task: Task }) {
  const router = useRouter();
  const { tenantId } = useTenant();
  const { push } = useToast();
  const [stage, setStage] = useState<"review" | "executing" | "interrupt">("review");
  const [rejectOpen, setRejectOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [budget, setBudget] = useState<BudgetConfig>({
    hardCapUsd: task.plan?.hard_cap_usd ?? 2.0,
    retryBudgetPerStep: task.plan?.retry_budget_per_step ?? 3,
    approveAndWalkAway: false,
  });

  const editForm = useForm<z.infer<typeof editsSchema>>({ resolver: zodResolver(editsSchema) });
  const rejectForm = useForm<{ reason: string }>();

  const [liveSteps, setLiveSteps] = useState<Step[]>([]);

  const sseUrl =
    stage === "executing" ? `${api.base}/v1/tenants/${tenantId}/tasks/${task.id}/events` : null;
  useSse<{ event_type: string; step?: Step; task?: Task }>(sseUrl, {
    onEvent: ({ data }) => {
      if (data.event_type === "task.step_started" && data.step) {
        setLiveSteps((p) => [...p, data.step as Step]);
      } else if (data.event_type === "task.step_completed" && data.step) {
        setLiveSteps((p) => p.map((s) => (s.step_id === data.step!.step_id ? data.step! : s)));
      } else if (data.event_type === "task.completed") {
        push({ title: "Task completed", description: "Verifier approved; promotion pending.", tone: "ok" });
        router.refresh();
      } else if (data.event_type === "task.failed") {
        push({ title: "Task failed", tone: "alert" });
        router.refresh();
      } else if (data.event_type === "task.budget_exceeded") {
        push({
          title: "Budget exceeded",
          description: "Agent halted. Replan or raise the cap.",
          tone: "alert",
        });
        setStage("interrupt");
      }
    },
  });

  if (!task.plan) {
    return (
      <Card>
        <CardContent className="p-6 text-sm text-muted-foreground">
          The planner is still working. This page will refresh when the plan is ready.
        </CardContent>
      </Card>
    );
  }

  const onApprove = async () => {
    setBusy(true);
    try {
      await api.approvePlan(tenantId, task.id, {
        retry_budget_per_step: budget.retryBudgetPerStep,
        hard_cap_usd: budget.hardCapUsd,
      });
      setStage("executing");
      push({
        title: "Plan approved",
        description: budget.approveAndWalkAway ? "Walk-away enabled — we'll handle the rest." : "Streaming progress.",
        tone: "ok",
      });
    } catch (e) {
      push({ title: "Approval failed", description: (e as Error).message, tone: "alert" });
    } finally {
      setBusy(false);
    }
  };

  const onSubmitEdits = async (v: z.infer<typeof editsSchema>) => {
    setBusy(true);
    try {
      await api.approvePlan(tenantId, task.id, {
        retry_budget_per_step: budget.retryBudgetPerStep,
        hard_cap_usd: budget.hardCapUsd,
        description: v.amendment ? `${task.plan!.description}\n\n[user amendment] ${v.amendment}` : task.plan!.description,
      });
      setStage("executing");
      push({ title: "Plan amended and approved", tone: "ok" });
    } catch (e) {
      push({ title: "Approval failed", description: (e as Error).message, tone: "alert" });
    } finally {
      setBusy(false);
    }
  };

  const onReject = async (v: { reason: string }) => {
    setBusy(true);
    try {
      await api.rejectPlan(tenantId, task.id, v.reason);
      push({ title: "Plan rejected", tone: "info" });
      router.push("/tasks");
    } catch (e) {
      push({ title: "Reject failed", description: (e as Error).message, tone: "alert" });
    } finally {
      setBusy(false);
      setRejectOpen(false);
    }
  };

  const onInterrupt = async () => {
    setBusy(true);
    try {
      await api.interruptTask(tenantId, task.id, "halt at next checkpoint");
      push({ title: "Halt requested", description: "Agent will stop at the next safe checkpoint.", tone: "warn" });
    } catch (e) {
      push({ title: "Interrupt failed", description: (e as Error).message, tone: "alert" });
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="grid grid-cols-[1fr_360px] gap-4">
      <div className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileSignature className="h-3.5 w-3.5" /> Plan
              <StatusBadge status={task.status} />
            </CardTitle>
            <CardDescription>
              Pre-execution preview. Approve, edit, or reject. The agent does not start until you sign.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <PlanSummary plan={task.plan as Plan} />
          </CardContent>
        </Card>

        {stage === "review" && (
          <Card>
            <CardHeader>
              <CardTitle>Amend the plan (optional)</CardTitle>
              <CardDescription>Provide notes the planner should incorporate before execution.</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={editForm.handleSubmit(onSubmitEdits)} className="space-y-3">
                <div className="space-y-1">
                  <Label htmlFor="amendment">Amendment</Label>
                  <Textarea
                    id="amendment"
                    rows={4}
                    placeholder="e.g. keep the existing error envelope; do not introduce a new feature flag"
                    {...editForm.register("amendment")}
                  />
                </div>
                <Button type="submit" variant="paper" disabled={busy}>
                  <MessagesSquare className="h-3.5 w-3.5" /> Approve with amendment
                </Button>
              </form>
            </CardContent>
          </Card>
        )}

        {stage !== "review" && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                Execution
                <StatusBadge status={task.status === "plan_approved" ? "executing" : task.status} />
              </CardTitle>
              <CardDescription>Live progress streamed via Server-Sent Events from the control plane.</CardDescription>
            </CardHeader>
            <CardContent>
              <StepTimeline steps={liveSteps.length > 0 ? liveSteps : task.steps ?? []} />
            </CardContent>
          </Card>
        )}
      </div>

      <aside className="space-y-3">
        <BudgetViewer
          estimatedCostUsd={task.plan.estimated_cost_usd}
          defaultHardCap={task.plan.hard_cap_usd}
          defaultRetryBudget={task.plan.retry_budget_per_step}
          onChange={setBudget}
        />

        <Card>
          <CardHeader>
            <CardTitle>Task identity</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-xs">
            <Row label="task" value={<HashPill value={task.id} />} />
            <Row label="repo" value={<span className="font-mono">{task.repo}</span>} />
            {task.base_sha && <Row label="base" value={<HashPill value={task.base_sha} />} />}
            <Row label="submitted by" value={task.submitted_by} />
            <Row label="source" value={<span className="font-mono">{task.submitted_from ?? "web"}</span>} />
          </CardContent>
        </Card>

        {stage === "review" ? (
          <div className="sticky bottom-4 flex flex-col gap-2 border border-ink-300 bg-background p-3 shadow-ink-lg dark:border-ink-700">
            <Button onClick={onApprove} disabled={busy} size="lg">
              Approve plan
            </Button>
            <Button variant="outline" disabled={busy} asChild>
              <Link href={`/tasks/${task.id}`}>Back to task</Link>
            </Button>
            <Button variant="destructive" onClick={() => setRejectOpen(true)} disabled={busy}>
              Reject
            </Button>
          </div>
        ) : (
          <div className="sticky bottom-4 flex flex-col gap-2 border border-accent-warn bg-background p-3 shadow-ink-lg">
            <Button variant="destructive" onClick={onInterrupt} disabled={busy} size="lg">
              <CircleStop className="h-4 w-4" /> Halt at next checkpoint
            </Button>
            <p className="text-xs text-muted-foreground">
              Interrupting is cooperative — the executor finishes its current shell command, persists state, then stops.
              No partial diffs are merged.
            </p>
          </div>
        )}
      </aside>

      <Dialog open={rejectOpen} onOpenChange={setRejectOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reject plan</DialogTitle>
            <DialogDescription>
              Reason is recorded in the task's attestation chain and shared with the planner for next iteration.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={rejectForm.handleSubmit(onReject)}>
            <div className="space-y-1 p-4">
              <Label htmlFor="reason">Reason</Label>
              <Textarea id="reason" rows={4} required {...rejectForm.register("reason", { required: true })} />
            </div>
            <DialogFooter>
              <Button variant="outline" type="button" onClick={() => setRejectOpen(false)}>
                Cancel
              </Button>
              <Button variant="destructive" type="submit" disabled={busy}>
                Reject plan
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-2">
      <span className="font-mono text-[10px] uppercase tracking-wide text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate text-right">{value}</span>
    </div>
  );
}
