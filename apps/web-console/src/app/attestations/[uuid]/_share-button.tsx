"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toaster";
import { Copy } from "lucide-react";

export function ShareButton({ rekorUuid }: { rekorUuid: string }) {
  const { push } = useToast();
  const [url, setUrl] = useState<string | null>(null);

  const gen = async () => {
    // In production: api.createAttestationShareLink(rekorUuid). For the
    // demo, synthesize a stable URL so the UI is reviewable end-to-end.
    const link = `https://attest.crucible.dev/share/${encodeURIComponent(rekorUuid)}?exp=${
      Math.floor(Date.now() / 1000) + 30 * 86_400
    }&sig=demo`;
    setUrl(link);
    push({ title: "Share link created", description: "Expires in 30 days.", tone: "ok" });
  };

  if (!url) {
    return (
      <Button variant="outline" className="w-full" onClick={gen}>
        Create share link
      </Button>
    );
  }
  return (
    <div className="space-y-2">
      <Input readOnly value={url} className="font-mono text-xs" />
      <Button
        variant="outline"
        className="w-full"
        onClick={async () => {
          await navigator.clipboard.writeText(url);
          push({ title: "Copied", tone: "info" });
        }}
      >
        <Copy className="h-3.5 w-3.5" /> Copy
      </Button>
    </div>
  );
}
