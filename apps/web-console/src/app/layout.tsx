import type { Metadata } from "next";
import { Toaster } from "@/components/ui/toaster";
import { SiteShell } from "@/components/site-shell";
import { TenantContextProvider } from "@/lib/tenant-context";
import "./globals.css";

export const metadata: Metadata = {
  title: { default: "Crucible", template: "%s · Crucible" },
  description:
    "Evidence-driven AI engineering. Plan, verify, attest. Built for senior engineers who scrutinize what the agent did.",
  robots: { index: false, follow: false },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <TenantContextProvider>
          <SiteShell>{children}</SiteShell>
          <Toaster />
        </TenantContextProvider>
      </body>
    </html>
  );
}
