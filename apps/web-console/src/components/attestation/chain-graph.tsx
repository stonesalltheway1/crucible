"use client";

import Link from "next/link";
import { ArrowDown, FileSignature, ShieldCheck, FileEdit, FileCheck2, BoxSelect, Rocket, Stamp, Activity } from "lucide-react";
import { HashPill } from "@/components/hash-pill";
import { cn, formatRelative } from "@/lib/utils";

const ICON: Record<string, React.ElementType> = {
  "Plan/v1": FileSignature,
  "PlanApproval/v1": Stamp,
  "TwinFsWrite/v1": FileEdit,
  "TestRun/v1": FileCheck2,
  "VerifierApproval/v1": ShieldCheck,
  "PromotionBundle/v1": BoxSelect,
  "PromotionApproval/v1": Stamp,
  "KmsLease/v1": Stamp,
  "PromotionOutcome/v1": Rocket,
  "MemoryWrite/v1": Activity,
};

export type ChainNode = {
  rekor_uuid: string;
  predicate_type: string;
  signed_at: string;
  label: string;
};
export type Chain = {
  task_id: string;
  nodes: ChainNode[];
  edges: { from: string; to: string }[];
};

// The trust-narrative surface. Renders the full attestation chain for a task.
// Vertical, monospace-heavy, copyable Rekor UUIDs on every node.
export function ChainGraph({ chain }: { chain: Chain }) {
  if (chain.nodes.length === 0) {
    return (
      <div className="border border-dashed border-ink-300 p-6 text-center text-xs text-muted-foreground dark:border-ink-700">
        No attestations yet — the chain is built incrementally as the task runs.
      </div>
    );
  }
  return (
    <div className="space-y-0">
      {chain.nodes.map((n, i) => {
        const Icon = ICON[n.predicate_type] ?? Activity;
        const last = i === chain.nodes.length - 1;
        return (
          <div key={n.rekor_uuid}>
            <div className="flex items-start gap-3">
              <div
                className={cn(
                  "grid h-8 w-8 place-items-center border border-ink-300 bg-ink-50 dark:border-ink-700 dark:bg-ink-900",
                )}
              >
                <Icon className="h-4 w-4" />
              </div>
              <div className="min-w-0 flex-1 border border-ink-200 bg-card p-3 dark:border-ink-800">
                <div className="flex flex-wrap items-baseline justify-between gap-x-4">
                  <div className="min-w-0">
                    <div className="text-sm font-medium">{n.label}</div>
                    <div className="font-mono text-[10px] uppercase tracking-wide text-muted-foreground">
                      {n.predicate_type} · {formatRelative(n.signed_at)}
                    </div>
                  </div>
                  <HashPill value={n.rekor_uuid} href={`/attestations/${encodeURIComponent(n.rekor_uuid)}`} />
                </div>
              </div>
            </div>
            {!last && (
              <div className="flex h-4 items-center pl-4">
                <ArrowDown className="h-3 w-3 text-muted-foreground" />
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
