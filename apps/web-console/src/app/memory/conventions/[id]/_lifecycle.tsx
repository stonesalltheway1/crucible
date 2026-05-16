"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { useToast } from "@/components/ui/toaster";
import { api, type Convention } from "@/lib/api";
import { useTenant } from "@/lib/tenant-context";

const TRANSITIONS: Record<Convention["status"], Convention["status"][]> = {
  candidate: ["active", "superseded"],
  active: ["drifting", "superseded"],
  drifting: ["active", "superseded"],
  superseded: [],
};

export function ConventionLifecycle({
  conventionId,
  status,
}: {
  conventionId: string;
  status: Convention["status"];
}) {
  const router = useRouter();
  const { tenantId } = useTenant();
  const { push } = useToast();
  const [busy, setBusy] = useState(false);

  const next = TRANSITIONS[status];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Lifecycle</CardTitle>
        <CardDescription>
          Senior-engineer overrides. Status changes are attested into the memory chain so the distiller respects them on
          the next pass.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-2">
        {next.length === 0 && (
          <div className="text-xs text-muted-foreground">Terminal state — no further transitions.</div>
        )}
        {next.map((target) => (
          <Button
            key={target}
            variant={target === "superseded" ? "destructive" : "paper"}
            className="w-full"
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              try {
                await api.setConventionStatus(tenantId, conventionId, target);
                push({ title: `Marked ${target}`, tone: "ok" });
                router.refresh();
              } catch (e) {
                push({ title: "Update failed", description: (e as Error).message, tone: "alert" });
              } finally {
                setBusy(false);
              }
            }}
          >
            Mark {target}
          </Button>
        ))}
      </CardContent>
    </Card>
  );
}
