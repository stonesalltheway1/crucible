"use client";
import * as React from "react";
import * as ProgressPrimitive from "@radix-ui/react-progress";
import { cn } from "@/lib/utils";

export const Progress = React.forwardRef<
  React.ElementRef<typeof ProgressPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof ProgressPrimitive.Root> & { tone?: "ink" | "ok" | "warn" | "alert" }
>(({ className, value, tone = "ink", ...props }, ref) => {
  const fill =
    tone === "ok" ? "bg-accent-ok" : tone === "warn" ? "bg-accent-warn" : tone === "alert" ? "bg-accent-alert" : "bg-ink-900";
  return (
    <ProgressPrimitive.Root
      ref={ref}
      className={cn("relative h-2 w-full overflow-hidden bg-ink-100 dark:bg-ink-800", className)}
      {...props}
    >
      <ProgressPrimitive.Indicator
        className={cn("h-full transition-all", fill)}
        style={{ transform: `translateX(-${100 - (value || 0)}%)` }}
      />
    </ProgressPrimitive.Root>
  );
});
Progress.displayName = ProgressPrimitive.Root.displayName;
