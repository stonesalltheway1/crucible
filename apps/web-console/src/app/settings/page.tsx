"use client";

import { useState } from "react";
import { PageHeader } from "@/components/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Input, Textarea } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import { Switch } from "@/components/ui/switch";
import { useToast } from "@/components/ui/toaster";
import { formatUsd } from "@/lib/utils";

const DEFAULT_POLICY = `# Tenant policy override (Rego).
# This bundle is signed; mutations require an attestation cycle.
# Defaults from libs/policy/bundles/promotion_default.rego still apply
# unless explicitly relaxed here.
package crucible.promotion

import rego.v1

# Example: require a payments-leads approver on anything that touches billing/.
deny contains msg if {
  some f
  input.bundle.files_changed[f].path == "billing/"
  not "approved_by_payments_leads" in input.approvals
  msg := "billing changes require @payments-leads approval"
}
`;

export default function SettingsPage() {
  const { push } = useToast();
  const [costCap, setCostCap] = useState(2.0);
  const [retryCap, setRetryCap] = useState(3);
  const [perDayBudget, setPerDayBudget] = useState(150);
  const [allowApproveAndWalkAway, setAllowApproveAndWalkAway] = useState(true);
  const [models, setModels] = useState({
    executor: "claude-opus-4-7",
    verifier: "gemini-3.1-pro",
    distiller: "claude-haiku-4-5",
  });
  const [policy, setPolicy] = useState(DEFAULT_POLICY);

  const save = () => push({ title: "Saved", description: "Tenant config updated. New tasks pick up the change.", tone: "ok" });

  return (
    <>
      <PageHeader
        title="Settings"
        description="Tenant-scoped configuration. Changes here are signed into the tenant config attestation chain."
      />

      <Tabs defaultValue="budgets">
        <TabsList>
          <TabsTrigger value="budgets">Budgets</TabsTrigger>
          <TabsTrigger value="models">Models</TabsTrigger>
          <TabsTrigger value="classifier">Critical-path</TabsTrigger>
          <TabsTrigger value="policy">Promotion policy</TabsTrigger>
        </TabsList>

        <TabsContent value="budgets" className="pt-4 space-y-3">
          <Card>
            <CardHeader>
              <CardTitle>Hard caps</CardTitle>
              <CardDescription>
                Defaults applied to new tasks. The per-task cap is overrideable per task; the per-day cap is enforced at
                the gateway.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div>
                <div className="flex items-baseline justify-between">
                  <Label>Per-task cap</Label>
                  <span className="font-mono text-sm tabular-nums">{formatUsd(costCap)}</span>
                </div>
                <Slider min={0.25} max={20} step={0.25} value={[costCap]} onValueChange={([v]) => setCostCap(v)} />
              </div>
              <div>
                <div className="flex items-baseline justify-between">
                  <Label>Per-day cap</Label>
                  <span className="font-mono text-sm tabular-nums">{formatUsd(perDayBudget)}</span>
                </div>
                <Slider min={10} max={2000} step={10} value={[perDayBudget]} onValueChange={([v]) => setPerDayBudget(v)} />
              </div>
              <div>
                <div className="flex items-baseline justify-between">
                  <Label>Retries per subgoal</Label>
                  <span className="font-mono text-sm tabular-nums">{retryCap}</span>
                </div>
                <Slider min={0} max={8} step={1} value={[retryCap]} onValueChange={([v]) => setRetryCap(v)} />
              </div>
              <div className="flex items-start gap-3 border-t border-ink-200 pt-4 dark:border-ink-800">
                <Switch checked={allowApproveAndWalkAway} onCheckedChange={setAllowApproveAndWalkAway} />
                <div>
                  <Label>Allow "approve and walk away"</Label>
                  <p className="text-xs text-muted-foreground">
                    Skip the verifier-result re-confirmation for non-critical-path tasks.
                  </p>
                </div>
              </div>
              <Button onClick={save}>Save</Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="models" className="pt-4">
          <Card>
            <CardHeader>
              <CardTitle>Model overrides</CardTitle>
              <CardDescription>Default routing picks per tier; override per-role when you have a strong preference.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {Object.entries(models).map(([role, val]) => (
                <div key={role}>
                  <Label>{role}</Label>
                  <Input
                    value={val}
                    className="font-mono"
                    onChange={(e) => setModels((m) => ({ ...m, [role]: e.target.value }))}
                  />
                </div>
              ))}
              <Button onClick={save}>Save</Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="classifier" className="pt-4">
          <Card>
            <CardHeader>
              <CardTitle>Critical-path classifier weights</CardTitle>
              <CardDescription>
                Per-tenant calibration of which files / globs the classifier treats as critical (escalates to Tier 3,
                requires N-of-M approval). Calibration runs via `crucible calibrate`.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {[
                { k: "billing/**", v: 0.95 },
                { k: "api/webhooks/**", v: 0.9 },
                { k: "auth/**", v: 0.92 },
                { k: "lib/orders/**", v: 0.7 },
                { k: "docs/**", v: 0.1 },
              ].map((w) => (
                <div key={w.k} className="grid grid-cols-[1fr_3fr_auto] items-center gap-3">
                  <span className="font-mono text-xs">{w.k}</span>
                  <Slider min={0} max={1} step={0.01} defaultValue={[w.v]} />
                  <span className="font-mono text-xs tabular-nums w-12 text-right">{w.v.toFixed(2)}</span>
                </div>
              ))}
              <Button onClick={save}>Save</Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="policy" className="pt-4">
          <Card>
            <CardHeader>
              <CardTitle>Promotion policy (Rego)</CardTitle>
              <CardDescription>
                Signed-bundle override on top of the default Crucible policy. Validated server-side against the policy
                schema before it lands.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <Textarea
                rows={20}
                className="font-mono text-xs"
                value={policy}
                onChange={(e) => setPolicy(e.target.value)}
              />
              <div className="flex gap-2">
                <Button onClick={save}>Save and sign</Button>
                <Button variant="outline" onClick={() => setPolicy(DEFAULT_POLICY)}>
                  Reset
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </>
  );
}
