"use client";
import * as React from "react";
import * as SwitchPrimitive from "@radix-ui/react-switch";
import { cn } from "@/lib/utils";

export const Switch = React.forwardRef<
  React.ElementRef<typeof SwitchPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SwitchPrimitive.Root>
>(({ className, ...props }, ref) => (
  <SwitchPrimitive.Root
    ref={ref}
    className={cn(
      "peer inline-flex h-5 w-9 shrink-0 cursor-pointer items-center border border-ink-300 transition-colors",
      "data-[state=checked]:bg-ink-900 data-[state=unchecked]:bg-ink-100",
      "dark:border-ink-700 dark:data-[state=checked]:bg-ink-100 dark:data-[state=unchecked]:bg-ink-800",
      className,
    )}
    {...props}
  >
    <SwitchPrimitive.Thumb className="pointer-events-none block h-4 w-4 bg-background shadow ring-0 transition-transform data-[state=checked]:translate-x-[18px] data-[state=unchecked]:translate-x-[2px]" />
  </SwitchPrimitive.Root>
));
Switch.displayName = SwitchPrimitive.Root.displayName;
