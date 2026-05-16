import { AlertOctagon, AlertTriangle, Circle, Database, Globe, Wallet, Clock, FileEdit } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { formatUsd } from "@/lib/utils";
import type { Plan } from "@/lib/api";

const RISK_TONE = { low: "info", med: "warn", high: "alert" } as const;
const RISK_ICON = { low: Circle, med: AlertTriangle, high: AlertOctagon };

export function PlanSummary({ plan }: { plan: Plan }) {
  return (
    <div className="space-y-4">
      <p className="text-sm leading-relaxed text-foreground">{plan.description}</p>

      <div className="grid grid-cols-4 gap-2">
        <Stat icon={Wallet} label="cost est." value={formatUsd(plan.estimated_cost_usd)} />
        <Stat icon={Clock} label="duration est." value={`~${plan.estimated_duration_min}m`} />
        <Stat icon={FileEdit} label="files" value={String(plan.files_to_touch.length)} />
        <Stat
          icon={Database}
          label="migrations"
          value={String(plan.db_migrations)}
          tone={plan.db_migrations > 0 ? "warn" : "mute"}
        />
      </div>

      <Section title="Files to touch">
        {plan.files_to_touch.length === 0 ? (
          <div className="text-xs text-muted-foreground">none</div>
        ) : (
          <ul className="flex flex-wrap gap-1 font-mono text-xs">
            {plan.files_to_touch.map((f) => (
              <li key={f} className="border border-ink-200 bg-ink-50 px-1.5 py-0.5 dark:border-ink-800 dark:bg-ink-900">
                {f}
              </li>
            ))}
          </ul>
        )}
      </Section>

      <Section title="External effects">
        {plan.external_effects.length === 0 ? (
          <div className="text-xs text-muted-foreground">none — all calls hit the twin's recorded tapes</div>
        ) : (
          <ul className="space-y-1">
            {plan.external_effects.map((eff, i) => (
              <li key={`${eff.service}-${i}`} className="flex items-center gap-2 text-xs">
                <Globe className="h-3 w-3 text-muted-foreground" />
                <span className="font-mono">{eff.service}</span>
                <span className="text-muted-foreground">→</span>
                <span className="font-mono">{eff.endpoints.join(", ")}</span>
                <Badge tone={eff.live ? "alert" : "mute"}>{eff.live ? "live" : "tape"}</Badge>
              </li>
            ))}
          </ul>
        )}
      </Section>

      <Section title="Top risks">
        {plan.top_risks.length === 0 ? (
          <div className="text-xs text-muted-foreground">none flagged by the planner</div>
        ) : (
          <ul className="space-y-2">
            {plan.top_risks.map((r, i) => {
              const Icon = RISK_ICON[r.impact];
              return (
                <li key={i} className="flex items-start gap-2 border border-ink-200 bg-ink-50 p-2 dark:border-ink-800 dark:bg-ink-900">
                  <Icon
                    className={`mt-0.5 h-3.5 w-3.5 ${
                      r.impact === "high"
                        ? "text-accent-alert"
                        : r.impact === "med"
                          ? "text-accent-warn"
                          : "text-muted-foreground"
                    }`}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm">{r.description}</span>
                      <Badge tone={RISK_TONE[r.impact]}>{r.impact}</Badge>
                    </div>
                    {r.mitigation && <div className="mt-1 text-xs text-muted-foreground">mitigation: {r.mitigation}</div>}
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </Section>
    </div>
  );
}

function Stat({
  icon: Icon,
  label,
  value,
  tone,
}: {
  icon: React.ElementType;
  label: string;
  value: string;
  tone?: "ok" | "warn" | "alert" | "mute";
}) {
  return (
    <div className="border border-ink-200 bg-card p-2 dark:border-ink-800">
      <div className="flex items-center gap-1 font-mono text-[10px] uppercase tracking-wide text-muted-foreground">
        <Icon className="h-3 w-3" /> {label}
      </div>
      <div
        className={`mt-0.5 text-lg font-semibold tabular-nums ${
          tone === "warn" ? "text-accent-warn" : tone === "alert" ? "text-accent-alert" : ""
        }`}
      >
        {value}
      </div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <div className="font-mono text-[10px] uppercase tracking-wide text-muted-foreground">{title}</div>
      {children}
    </div>
  );
}
