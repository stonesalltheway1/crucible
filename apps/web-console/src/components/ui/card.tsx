import * as React from "react";
import { cn } from "@/lib/utils";

const Card = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...p }, ref) => (
    <div
      ref={ref}
      className={cn(
        "rounded-none border border-ink-200 bg-card text-card-foreground shadow-ink dark:border-ink-800",
        className,
      )}
      {...p}
    />
  ),
);
Card.displayName = "Card";

const CardHeader = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...p }, ref) => (
    <div ref={ref} className={cn("flex flex-col gap-1 p-4 border-b border-ink-200 dark:border-ink-800", className)} {...p} />
  ),
);
CardHeader.displayName = "CardHeader";

const CardTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(
  ({ className, ...p }, ref) => (
    <h3 ref={ref} className={cn("text-sm font-semibold tracking-tight text-ink-900 dark:text-ink-100", className)} {...p} />
  ),
);
CardTitle.displayName = "CardTitle";

const CardDescription = React.forwardRef<HTMLParagraphElement, React.HTMLAttributes<HTMLParagraphElement>>(
  ({ className, ...p }, ref) => <p ref={ref} className={cn("text-xs text-muted-foreground", className)} {...p} />,
);
CardDescription.displayName = "CardDescription";

const CardContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...p }, ref) => <div ref={ref} className={cn("p-4", className)} {...p} />,
);
CardContent.displayName = "CardContent";

const CardFooter = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...p }, ref) => (
    <div ref={ref} className={cn("flex items-center gap-2 px-4 py-3 border-t border-ink-200 dark:border-ink-800", className)} {...p} />
  ),
);
CardFooter.displayName = "CardFooter";

export { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter };
