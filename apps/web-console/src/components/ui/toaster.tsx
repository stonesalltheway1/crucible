"use client";

import * as React from "react";
import * as ToastPrimitive from "@radix-ui/react-toast";
import { cn } from "@/lib/utils";

type Toast = { id: string; title: string; description?: string; tone?: "info" | "ok" | "warn" | "alert" };
const ToastCtx = React.createContext<{ push: (t: Omit<Toast, "id">) => void } | undefined>(undefined);

export function Toaster() {
  const [toasts, setToasts] = React.useState<Toast[]>([]);
  const push = React.useCallback((t: Omit<Toast, "id">) => {
    const id = Math.random().toString(36).slice(2);
    setToasts((p) => [...p, { id, ...t }]);
    setTimeout(() => setToasts((p) => p.filter((x) => x.id !== id)), 4500);
  }, []);
  return (
    <ToastCtx.Provider value={{ push }}>
      <ToastPrimitive.Provider swipeDirection="right">
        {toasts.map((t) => (
          <ToastPrimitive.Root
            key={t.id}
            className={cn(
              "grid grid-cols-[1fr_auto] gap-2 border bg-background p-3 shadow-ink-lg animate-slide-in-bottom",
              t.tone === "alert" && "border-accent-alert",
              t.tone === "ok" && "border-accent-ok",
              t.tone === "warn" && "border-accent-warn",
              !t.tone && "border-ink-300 dark:border-ink-700",
            )}
          >
            <div>
              <ToastPrimitive.Title className="text-sm font-medium">{t.title}</ToastPrimitive.Title>
              {t.description && (
                <ToastPrimitive.Description className="text-xs text-muted-foreground">
                  {t.description}
                </ToastPrimitive.Description>
              )}
            </div>
          </ToastPrimitive.Root>
        ))}
        <ToastPrimitive.Viewport className="fixed bottom-4 right-4 z-[60] flex w-[360px] max-w-[100vw] flex-col gap-2" />
      </ToastPrimitive.Provider>
    </ToastCtx.Provider>
  );
}

export function useToast() {
  const ctx = React.useContext(ToastCtx);
  if (!ctx) return { push: (_t: Omit<Toast, "id">) => {} };
  return ctx;
}
