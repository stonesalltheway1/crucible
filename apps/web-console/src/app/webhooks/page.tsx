"use client";

import { useState } from "react";
import { PageHeader } from "@/components/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useToast } from "@/components/ui/toaster";
import { HashPill } from "@/components/hash-pill";
import { Webhook, Plus, RefreshCcw } from "lucide-react";

const EVENT_GLOBS = [
  "task.*",
  "task.promotion_*",
  "memory.convention_drift_detected",
  "memory.convention_learned",
  "security.*",
  "system.*",
];

export default function WebhooksPage() {
  const { push } = useToast();
  const [subs, setSubs] = useState([
    {
      id: "sub_h7y3n2",
      url: "https://hooks.acme.com/crucible",
      events: ["task.*", "memory.convention_drift_detected"],
      active: true,
      created_at: "2026-04-12",
      last_delivery: "1m ago",
      last_status: "200",
    },
    {
      id: "sub_xa4qz1",
      url: "https://eng-bot.acme.internal/intake",
      events: ["security.*"],
      active: true,
      created_at: "2026-03-04",
      last_delivery: "1h ago",
      last_status: "200",
    },
  ]);
  const [url, setUrl] = useState("");
  const [events, setEvents] = useState<string[]>(["task.*"]);

  const create = () => {
    if (!url.startsWith("https://")) {
      push({ title: "URL must be https://", tone: "alert" });
      return;
    }
    const id = `sub_${Math.random().toString(36).slice(2, 9)}`;
    setSubs((s) => [...s, { id, url, events, active: true, created_at: "now", last_delivery: "—", last_status: "—" }]);
    push({
      title: "Subscription created",
      description: "Signing secret is shown once — copy it now.",
      tone: "ok",
    });
    setUrl("");
  };

  return (
    <>
      <PageHeader
        title="Webhooks"
        description="Subscribe to Crucible events. Every payload is HMAC-signed; high-stakes events additionally carry a Sigstore-keyless bundle."
      />

      <Card className="mb-4">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Plus className="h-3.5 w-3.5" /> New subscription
          </CardTitle>
          <CardDescription>
            Receiver must verify the `X-Crucible-Signature` HMAC header before processing. We refuse to mint a
            subscription that doesn't accept signed payloads in setup verification.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div>
            <Label>Endpoint URL</Label>
            <Input
              placeholder="https://hooks.example.com/crucible"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              className="font-mono"
            />
          </div>
          <div>
            <Label>Events</Label>
            <div className="mt-1 flex flex-wrap gap-1">
              {EVENT_GLOBS.map((e) => {
                const on = events.includes(e);
                return (
                  <button
                    key={e}
                    type="button"
                    onClick={() => setEvents((cur) => (on ? cur.filter((x) => x !== e) : [...cur, e]))}
                    className={`border px-2 py-0.5 font-mono text-xs ${
                      on ? "border-ink-900 bg-ink-900 text-ink-50" : "border-ink-300 hover:bg-ink-100 dark:border-ink-700"
                    }`}
                  >
                    {e}
                  </button>
                );
              })}
            </div>
          </div>
          <Button onClick={create}>Create</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Webhook className="h-3.5 w-3.5" /> Active subscriptions
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Endpoint</TableHead>
                <TableHead>Events</TableHead>
                <TableHead>Last delivery</TableHead>
                <TableHead>Status</TableHead>
                <TableHead></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {subs.map((s) => (
                <TableRow key={s.id}>
                  <TableCell>
                    <HashPill value={s.id} />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{s.url}</TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {s.events.map((e) => (
                        <Badge key={e} tone="info">{e}</Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">{s.last_delivery}</TableCell>
                  <TableCell>
                    <Badge tone={s.last_status === "200" ? "ok" : "warn"}>{s.last_status}</Badge>
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => push({ title: "Redelivery queued", tone: "info" })}
                    >
                      <RefreshCcw className="h-3 w-3" /> Redeliver
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </>
  );
}
