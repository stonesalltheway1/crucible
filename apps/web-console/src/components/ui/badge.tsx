import * as React from "react";
import { cn } from "@/lib/utils";

type Tone = "ok" | "warn" | "alert" | "info" | "mute";

export function Badge({
  tone = "mute",
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLSpanElement> & { tone?: Tone }) {
  return (
    <span className={cn("pill", `pill-${tone}`, className)} {...props}>
      {children}
    </span>
  );
}
