"use client";

import { createContext, useContext, useMemo, useState } from "react";

// In real deployments, tenantId comes from the Clerk JWT (`org_id` claim)
// or the WorkOS session's `organization` field. This provider centralizes
// access so server-rendered routes and client components share one source.
type TenantContextValue = {
  tenantId: string;
  tenantName: string;
  role: "owner" | "approver" | "developer" | "viewer";
  setTenantId: (id: string) => void;
};

const Ctx = createContext<TenantContextValue | undefined>(undefined);

export function TenantContextProvider({ children }: { children: React.ReactNode }) {
  const [tenantId, setTenantId] = useState<string>(
    process.env.NEXT_PUBLIC_DEFAULT_TENANT_ID || "ten_demo",
  );
  const value = useMemo<TenantContextValue>(
    () => ({
      tenantId,
      tenantName: "Acme Payments",
      role: "approver",
      setTenantId,
    }),
    [tenantId],
  );
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useTenant(): TenantContextValue {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useTenant must be used inside <TenantContextProvider>");
  return ctx;
}
