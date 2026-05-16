"use client";

import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

// Crucible button — anti-vibe variants. Default is `ink` (solid charcoal),
// no glow, sharp corners, no gradient. Documents-over-pills aesthetic.
const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        ink: "border border-ink-800 bg-ink-900 text-ink-50 hover:bg-ink-800",
        paper:
          "border border-ink-300 bg-ink-50 text-ink-900 hover:bg-ink-100 dark:border-ink-700 dark:bg-ink-900 dark:text-ink-100 dark:hover:bg-ink-800",
        ghost:
          "text-ink-700 hover:bg-ink-100 dark:text-ink-200 dark:hover:bg-ink-800",
        outline:
          "border border-ink-300 bg-transparent text-ink-900 hover:bg-ink-100 dark:border-ink-700 dark:text-ink-100 dark:hover:bg-ink-800",
        destructive:
          "border border-accent-alert bg-transparent text-accent-alert hover:bg-accent-alert hover:text-ink-50",
        link: "text-ink-900 underline-offset-4 hover:underline dark:text-ink-100",
      },
      size: {
        sm: "h-7 px-2.5 text-xs",
        md: "h-8 px-3",
        lg: "h-10 px-4 text-base",
        icon: "h-8 w-8",
      },
    },
    defaultVariants: { variant: "ink", size: "md" },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return <Comp className={cn(buttonVariants({ variant, size, className }))} ref={ref} {...props} />;
  },
);
Button.displayName = "Button";

export { buttonVariants };
