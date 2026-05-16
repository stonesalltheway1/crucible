import { cn, formatDuration, formatRelative } from "@/lib/utils";
import { Check, Circle, X, CircleDot } from "lucide-react";

type Step = {
  weight: number;
  dwell_seconds: number;
  slo_check: "pending" | "passed" | "failed";
  started_at?: string;
};

export function CanaryStrip({ steps, current }: { steps: Step[]; current: number }) {
  return (
    <ol className="flex gap-0">
      {steps.map((s, i) => {
        const past = i < current;
        const active = i === current;
        const tone =
          s.slo_check === "passed"
            ? "ok"
            : s.slo_check === "failed"
              ? "alert"
              : active
                ? "info"
                : "mute";
        const Icon = s.slo_check === "passed" ? Check : s.slo_check === "failed" ? X : active ? CircleDot : Circle;
        const cls = {
          ok: "border-accent-ok text-accent-ok",
          alert: "border-accent-alert text-accent-alert",
          info: "border-accent-info text-accent-info",
          mute: "border-ink-300 text-muted-foreground",
        }[tone];
        return (
          <li key={i} className="flex-1 border-r border-ink-200 last:border-r-0 dark:border-ink-800">
            <div className={cn("flex items-center gap-2 border-l-4 p-3", cls)}>
              <Icon className="h-4 w-4" />
              <div className="min-w-0 flex-1">
                <div className="text-sm font-medium tabular-nums">{s.weight}%</div>
                <div className="font-mono text-[10px] uppercase text-muted-foreground">
                  dwell {formatDuration(s.dwell_seconds)} · slo {s.slo_check}
                </div>
                {s.started_at && (
                  <div className="font-mono text-[10px] text-muted-foreground">started {formatRelative(s.started_at)}</div>
                )}
              </div>
            </div>
          </li>
        );
      })}
    </ol>
  );
}
