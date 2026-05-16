"use client";

import { Check, Circle, CircleAlert, CircleDot, Square } from "lucide-react";
import { cn, formatDuration, formatUsd } from "@/lib/utils";
import type { Step } from "@/lib/api";

const ICON = {
  pending: { node: Square, cls: "text-muted-foreground" },
  running: { node: CircleDot, cls: "text-accent-info animate-pulse" },
  completed: { node: Check, cls: "text-accent-ok" },
  failed: { node: CircleAlert, cls: "text-accent-alert" },
  skipped: { node: Circle, cls: "text-muted-foreground" },
} as const;

export function StepTimeline({ steps }: { steps: Step[] }) {
  if (steps.length === 0) {
    return (
      <div className="border border-dashed border-ink-300 p-4 text-center text-xs text-muted-foreground dark:border-ink-700">
        Waiting for the executor to begin step 1.
      </div>
    );
  }
  return (
    <ol className="space-y-2">
      {steps.map((s, i) => {
        const { node: Icon, cls } = ICON[s.status];
        return (
          <li key={s.step_id}>
            <div className="flex items-start gap-3">
              <div className={cn("mt-0.5 grid h-5 w-5 place-items-center border border-ink-300 bg-background dark:border-ink-700", cls)}>
                <Icon className="h-3 w-3" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between">
                  <div className="text-sm">
                    <span className="font-mono text-[10px] text-muted-foreground">step {i + 1}</span>{" "}
                    <span className="font-medium">{s.name}</span>
                  </div>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground tabular-nums">
                    {s.cost_usd > 0 && <span>{formatUsd(s.cost_usd)}</span>}
                    {s.duration_seconds > 0 && <span>{formatDuration(s.duration_seconds)}</span>}
                  </div>
                </div>
                {s.files_changed.length > 0 && (
                  <div className="mt-1 flex flex-wrap gap-1">
                    {s.files_changed.map((f) => (
                      <span key={f} className="font-mono text-[10px] text-muted-foreground">
                        {f}
                      </span>
                    ))}
                  </div>
                )}
                {s.error && <div className="mt-1 text-xs text-accent-alert">{s.error}</div>}
              </div>
            </div>
          </li>
        );
      })}
    </ol>
  );
}
