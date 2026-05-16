"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useToast } from "@/components/ui/toaster";
import { api } from "@/lib/api";
import { CheckCircle2, ShieldCheck, X, Loader2 } from "lucide-react";

type State =
  | { kind: "idle" }
  | { kind: "checking" }
  | { kind: "ok"; details: { inclusion_proof_valid: boolean; cert_chain_valid: boolean; subject_digest_matches: boolean; self_hosted: boolean } }
  | { kind: "fail"; reason: string };

export function VerifyButton({ rekorUuid }: { rekorUuid: string }) {
  const [state, setState] = useState<State>({ kind: "idle" });
  const { push } = useToast();

  const run = async () => {
    setState({ kind: "checking" });
    try {
      const res = await api.verifyAttestation(rekorUuid);
      if (res.verified) {
        setState({ kind: "ok", details: res.details });
        push({ title: "Verified", description: "End-to-end chain valid.", tone: "ok" });
      } else {
        setState({ kind: "fail", reason: "verifier returned not-verified" });
        push({ title: "Verification failed", tone: "alert" });
      }
    } catch (e) {
      setState({ kind: "fail", reason: (e as Error).message });
      push({ title: "Verification failed", description: (e as Error).message, tone: "alert" });
    }
  };

  return (
    <div className="space-y-2">
      <Button className="w-full" onClick={run} disabled={state.kind === "checking"}>
        {state.kind === "checking" ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
        ) : (
          <ShieldCheck className="h-3.5 w-3.5" />
        )}
        Verify end-to-end
      </Button>
      {state.kind === "ok" && (
        <div className="space-y-1 text-xs">
          <Check label="inclusion proof" ok={state.details.inclusion_proof_valid} />
          <Check label="certificate chain" ok={state.details.cert_chain_valid} />
          <Check label="subject digest" ok={state.details.subject_digest_matches} />
        </div>
      )}
      {state.kind === "fail" && (
        <div className="text-xs text-accent-alert">{state.reason}</div>
      )}
    </div>
  );
}

function Check({ label, ok }: { label: string; ok: boolean }) {
  return (
    <div className="flex items-center justify-between">
      <span>{label}</span>
      {ok ? (
        <Badge tone="ok"><CheckCircle2 className="h-3 w-3" /> ok</Badge>
      ) : (
        <Badge tone="alert"><X className="h-3 w-3" /> fail</Badge>
      )}
    </div>
  );
}
