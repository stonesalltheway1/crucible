"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { useTenant } from "@/lib/tenant-context";
import {
  Activity,
  CircleCheckBig,
  FileCheck2,
  Gauge,
  KeyRound,
  Layers,
  ListChecks,
  ScrollText,
  Settings,
  ShieldCheck,
  Wallet,
  Webhook,
} from "lucide-react";

const NAV = [
  { href: "/", label: "Overview", icon: Activity },
  { href: "/tasks", label: "Tasks", icon: ListChecks },
  { href: "/promotions", label: "Promotions", icon: CircleCheckBig },
  { href: "/memory", label: "Memory", icon: Layers },
  { href: "/attestations", label: "Attestations", icon: ShieldCheck },
  { href: "/cost", label: "Cost", icon: Wallet },
  { href: "/slo", label: "SLO", icon: Gauge },
  { href: "/webhooks", label: "Webhooks", icon: Webhook },
  { href: "/settings", label: "Settings", icon: Settings },
] as const;

export function SiteShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { tenantName, role } = useTenant();

  return (
    <div className="grid min-h-screen grid-cols-[200px_1fr] bg-background text-foreground">
      <aside className="border-r border-ink-200 bg-ink-50 dark:border-ink-800 dark:bg-ink-900">
        <div className="flex h-14 items-center gap-2 border-b border-ink-200 px-4 dark:border-ink-800">
          <div className="grid h-7 w-7 place-items-center border border-ink-900 bg-ink-900 text-ink-50">
            <ScrollText className="h-3.5 w-3.5" />
          </div>
          <div className="leading-tight">
            <div className="text-sm font-semibold tracking-tight">Crucible</div>
            <div className="font-mono text-[10px] uppercase text-muted-foreground">evidence — not vibes</div>
          </div>
        </div>
        <nav className="flex flex-col py-2">
          {NAV.map(({ href, label, icon: Icon }) => {
            const active = href === "/" ? pathname === "/" : pathname.startsWith(href);
            return (
              <Link
                key={href}
                href={href}
                className={cn(
                  "mx-2 flex items-center gap-2 border-l-2 border-transparent px-2 py-1.5 text-sm",
                  active
                    ? "border-ink-900 bg-ink-100 text-ink-900 dark:border-ink-100 dark:bg-ink-800 dark:text-ink-50"
                    : "text-ink-700 hover:bg-ink-100 dark:text-ink-300 dark:hover:bg-ink-800",
                )}
              >
                <Icon className="h-3.5 w-3.5" />
                <span>{label}</span>
              </Link>
            );
          })}
        </nav>
      </aside>
      <main className="flex flex-col">
        <header className="flex h-14 items-center justify-between border-b border-ink-200 px-6 dark:border-ink-800">
          <div className="flex items-center gap-3">
            <FileCheck2 className="h-4 w-4 text-muted-foreground" />
            <div>
              <div className="text-sm font-semibold tracking-tight">{tenantName}</div>
              <div className="font-mono text-[10px] uppercase text-muted-foreground">tenant · role:{role}</div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <KeyRound className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="font-mono text-xs text-muted-foreground">signed-in via OIDC</span>
          </div>
        </header>
        <div className="flex-1 p-6">{children}</div>
      </main>
    </div>
  );
}
