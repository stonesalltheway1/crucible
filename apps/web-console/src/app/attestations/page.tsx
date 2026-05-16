"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { PageHeader } from "@/components/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ShieldCheck, ScanSearch, Share2 } from "lucide-react";
import { HashPill } from "@/components/hash-pill";

const RECENT = [
  { uuid: "rekor:b2cdd9f4c8a1a3e2", predicate: "VerifierApproval/v1", at: "2m ago" },
  { uuid: "rekor:a9eed1f0c0a5b4d3", predicate: "PromotionOutcome/v1", at: "4m ago" },
  { uuid: "rekor:74cdc8a3b9c2e1f7", predicate: "TwinFsWrite/v1", at: "11m ago" },
  { uuid: "rekor:18ba8f2710d6a0b3", predicate: "MemoryWrite/v1", at: "27m ago" },
  { uuid: "rekor:6ddabbc02390fe5a", predicate: "PromotionApproval/v1", at: "1h ago" },
];

export default function AttestationsPage() {
  const [q, setQ] = useState("");
  const router = useRouter();

  return (
    <>
      <PageHeader
        title="Attestations"
        description="Search the per-tenant attestation log by Rekor UUID, predicate type, task id, or subject digest. Every entry is verifiable end-to-end against the Sigstore trust root."
      />

      <Card className="mb-4">
        <CardContent className="p-4">
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (q.trim()) router.push(`/attestations/${encodeURIComponent(q.trim())}`);
            }}
            className="flex items-center gap-2"
          >
            <ScanSearch className="h-4 w-4 text-muted-foreground" />
            <Input
              autoFocus
              className="font-mono"
              placeholder="rekor:b2cdd9f4c8a1a3e2  ·  task_01HZ...  ·  sha256:..."
              value={q}
              onChange={(e) => setQ(e.target.value)}
            />
            <Button type="submit">Resolve</Button>
          </form>
        </CardContent>
      </Card>

      <div className="grid grid-cols-[1fr_320px] gap-4">
        <Card>
          <CardHeader>
            <CardTitle>Recent attestations</CardTitle>
            <CardDescription>The last entries published to the relay; tenant-scoped.</CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="divide-y divide-ink-200 dark:divide-ink-800">
              {RECENT.map((r) => (
                <li key={r.uuid} className="flex items-center justify-between py-2">
                  <div className="flex items-center gap-3">
                    <ShieldCheck className="h-3.5 w-3.5 text-muted-foreground" />
                    <div>
                      <Badge tone="info">{r.predicate}</Badge>
                      <div className="mt-1 font-mono text-[10px] text-muted-foreground">{r.at}</div>
                    </div>
                  </div>
                  <HashPill value={r.uuid} href={`/attestations/${encodeURIComponent(r.uuid)}`} />
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Share2 className="h-3.5 w-3.5" /> Verify offline
            </CardTitle>
            <CardDescription>For air-gap auditors. Generates a public-share link with a time-bounded signed URL.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-xs">
            <p>
              The verifier can validate inclusion proofs and certificate chains entirely client-side, without further
              backend round-trips beyond the initial fetch.
            </p>
            <Button variant="outline" className="w-full" disabled>
              Generate share link (select an attestation)
            </Button>
          </CardContent>
        </Card>
      </div>
    </>
  );
}
