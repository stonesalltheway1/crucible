"use client";

import { Progress } from "@/components/ui/progress";
import { Slider } from "@/components/ui/slider";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Wallet, RotateCcw, ShieldAlert } from "lucide-react";
import { useState } from "react";
import { clampPercent, formatUsd } from "@/lib/utils";

export type BudgetConfig = {
  hardCapUsd: number;
  retryBudgetPerStep: number;
  approveAndWalkAway: boolean;
};

// The brand-promise visualization: cost cap + retry cap + walk-away toggle.
// The shape of this surface is the "Crucible looks different" signature.
export function BudgetViewer({
  estimatedCostUsd,
  defaultHardCap,
  defaultRetryBudget,
  onChange,
}: {
  estimatedCostUsd: number;
  defaultHardCap: number;
  defaultRetryBudget: number;
  onChange?: (cfg: BudgetConfig) => void;
}) {
  const [hardCap, setHardCap] = useState(defaultHardCap);
  const [retryBudget, setRetryBudget] = useState(defaultRetryBudget);
  const [walkAway, setWalkAway] = useState(false);
  const used = clampPercent((estimatedCostUsd / Math.max(hardCap, 0.01)) * 100);

  const emit = (next: Partial<BudgetConfig>) => {
    const cfg: BudgetConfig = {
      hardCapUsd: next.hardCapUsd ?? hardCap,
      retryBudgetPerStep: next.retryBudgetPerStep ?? retryBudget,
      approveAndWalkAway: next.approveAndWalkAway ?? walkAway,
    };
    onChange?.(cfg);
  };

  return (
    <div className="space-y-4 border border-ink-200 bg-card p-4 dark:border-ink-800">
      <div className="flex items-center gap-2">
        <Wallet className="h-4 w-4" />
        <span className="font-mono text-[11px] uppercase tracking-wide text-muted-foreground">budget cap</span>
      </div>

      <div>
        <div className="mb-1 flex items-baseline justify-between">
          <div className="text-2xl font-semibold tabular-nums">{formatUsd(hardCap)}</div>
          <div className="font-mono text-xs text-muted-foreground">
            est. {formatUsd(estimatedCostUsd)} · {used.toFixed(0)}% of cap
          </div>
        </div>
        <Progress value={used} tone={used > 80 ? "warn" : "ink"} />
        <div className="mt-3">
          <Slider
            min={0.25}
            max={Math.max(20, defaultHardCap * 4)}
            step={0.25}
            value={[hardCap]}
            onValueChange={([v]) => {
              setHardCap(v);
              emit({ hardCapUsd: v });
            }}
          />
          <div className="mt-1 flex justify-between text-[10px] text-muted-foreground">
            <span>$0.25</span>
            <span>{formatUsd(Math.max(20, defaultHardCap * 4))}</span>
          </div>
        </div>
        <p className="mt-2 text-xs text-muted-foreground">
          Hard cap. Once breached, the agent halts and waits for a human; it never silently exceeds.
        </p>
      </div>

      <div className="border-t border-ink-200 pt-4 dark:border-ink-800">
        <div className="flex items-center gap-2">
          <RotateCcw className="h-4 w-4" />
          <span className="font-mono text-[11px] uppercase tracking-wide text-muted-foreground">retry budget</span>
        </div>
        <div className="mt-2 flex items-baseline justify-between">
          <div className="text-2xl font-semibold tabular-nums">{retryBudget}</div>
          <div className="font-mono text-xs text-muted-foreground">retries / subgoal</div>
        </div>
        <Slider
          className="mt-2"
          min={0}
          max={8}
          step={1}
          value={[retryBudget]}
          onValueChange={([v]) => {
            setRetryBudget(v);
            emit({ retryBudgetPerStep: v });
          }}
        />
        <p className="mt-2 text-xs text-muted-foreground">
          When retries are exhausted, the executor halts and asks. Set to 0 to disable retries entirely.
        </p>
      </div>

      <div className="flex items-start gap-3 border-t border-ink-200 pt-4 dark:border-ink-800">
        <Switch
          id="walk-away"
          checked={walkAway}
          onCheckedChange={(v) => {
            setWalkAway(v);
            emit({ approveAndWalkAway: v });
          }}
        />
        <div className="flex-1">
          <Label htmlFor="walk-away" className="cursor-pointer text-xs">
            <span className="flex items-center gap-1">
              <ShieldAlert className="h-3 w-3" /> Approve and walk away
            </span>
          </Label>
          <p className="mt-0.5 text-xs text-muted-foreground">
            Skip the verifier-result re-confirmation. Promotion still goes through the gate; this only auto-acks the
            verifier success step. Available for tasks below the critical-path threshold.
          </p>
        </div>
      </div>
    </div>
  );
}
