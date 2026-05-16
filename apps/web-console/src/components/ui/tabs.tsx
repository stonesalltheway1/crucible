"use client";
import * as React from "react";
import * as TabsPrimitive from "@radix-ui/react-tabs";
import { cn } from "@/lib/utils";

export const Tabs = TabsPrimitive.Root;

export const TabsList = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.List>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.List>
>(({ className, ...p }, ref) => (
  <TabsPrimitive.List
    ref={ref}
    className={cn(
      "inline-flex items-center gap-0 border-b border-ink-200 dark:border-ink-800",
      className,
    )}
    {...p}
  />
));
TabsList.displayName = TabsPrimitive.List.displayName;

export const TabsTrigger = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger>
>(({ className, ...p }, ref) => (
  <TabsPrimitive.Trigger
    ref={ref}
    className={cn(
      "inline-flex items-center gap-2 border-b-2 border-transparent px-3 py-1.5 text-sm font-medium text-muted-foreground",
      "data-[state=active]:border-ink-900 data-[state=active]:text-ink-900 dark:data-[state=active]:border-ink-100 dark:data-[state=active]:text-ink-100",
      "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
      className,
    )}
    {...p}
  />
));
TabsTrigger.displayName = TabsPrimitive.Trigger.displayName;

export const TabsContent = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Content>
>(({ className, ...p }, ref) => (
  <TabsPrimitive.Content
    ref={ref}
    className={cn("focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring", className)}
    {...p}
  />
));
TabsContent.displayName = TabsPrimitive.Content.displayName;
