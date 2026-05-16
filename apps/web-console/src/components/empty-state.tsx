import type { LucideIcon } from "lucide-react";

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
}: {
  icon: LucideIcon;
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center border border-dashed border-ink-300 p-10 text-center dark:border-ink-700">
      <Icon className="mb-3 h-6 w-6 text-muted-foreground" />
      <div className="text-sm font-semibold">{title}</div>
      {description && <div className="mt-1 max-w-md text-xs text-muted-foreground">{description}</div>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}
