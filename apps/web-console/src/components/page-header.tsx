import { cn } from "@/lib/utils";

export function PageHeader({
  title,
  description,
  actions,
  className,
}: {
  title: string;
  description?: string;
  actions?: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("mb-6 flex items-end justify-between gap-4", className)}>
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        {description && <p className="max-w-2xl text-sm text-muted-foreground">{description}</p>}
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  );
}

export function MetricStat({
  label,
  value,
  hint,
  tone,
}: {
  label: string;
  value: React.ReactNode;
  hint?: React.ReactNode;
  tone?: "ok" | "warn" | "alert" | "info";
}) {
  const toneCls =
    tone === "ok"
      ? "text-accent-ok"
      : tone === "warn"
        ? "text-accent-warn"
        : tone === "alert"
          ? "text-accent-alert"
          : tone === "info"
            ? "text-accent-info"
            : "text-foreground";
  return (
    <div className="border border-ink-200 bg-card p-4 dark:border-ink-800">
      <div className="font-mono text-[10px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className={cn("mt-1 text-2xl font-semibold tabular-nums tracking-tight", toneCls)}>{value}</div>
      {hint && <div className="mt-1 text-xs text-muted-foreground">{hint}</div>}
    </div>
  );
}
