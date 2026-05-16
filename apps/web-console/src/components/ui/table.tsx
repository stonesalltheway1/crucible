import * as React from "react";
import { cn } from "@/lib/utils";

export const Table = React.forwardRef<HTMLTableElement, React.HTMLAttributes<HTMLTableElement>>(
  ({ className, ...p }, ref) => (
    <div className="w-full overflow-auto">
      <table ref={ref} className={cn("w-full caption-bottom text-sm tabular-nums", className)} {...p} />
    </div>
  ),
);
Table.displayName = "Table";

export const TableHeader = ({ className, ...p }: React.HTMLAttributes<HTMLTableSectionElement>) => (
  <thead className={cn("border-b border-ink-200 bg-ink-50 dark:border-ink-800 dark:bg-ink-900", className)} {...p} />
);
export const TableBody = ({ className, ...p }: React.HTMLAttributes<HTMLTableSectionElement>) => (
  <tbody className={cn("divide-y divide-ink-200 dark:divide-ink-800", className)} {...p} />
);
export const TableRow = ({ className, ...p }: React.HTMLAttributes<HTMLTableRowElement>) => (
  <tr className={cn("hover:bg-ink-50/60 dark:hover:bg-ink-900/60", className)} {...p} />
);
export const TableHead = ({ className, ...p }: React.ThHTMLAttributes<HTMLTableCellElement>) => (
  <th className={cn("h-9 px-3 text-left align-middle text-[11px] uppercase tracking-wide text-muted-foreground", className)} {...p} />
);
export const TableCell = ({ className, ...p }: React.TdHTMLAttributes<HTMLTableCellElement>) => (
  <td className={cn("px-3 py-2 align-middle", className)} {...p} />
);
