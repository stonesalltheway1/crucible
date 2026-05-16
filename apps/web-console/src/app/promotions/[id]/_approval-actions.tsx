"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useToast } from "@/components/ui/toaster";
import { api } from "@/lib/api";
import { useTenant } from "@/lib/tenant-context";
import { useRouter } from "next/navigation";

export function ApprovalActions({
  promotionId,
  status,
  bundleHash,
}: {
  promotionId: string;
  status: string;
  bundleHash: string;
}) {
  const router = useRouter();
  const { tenantId } = useTenant();
  const { push } = useToast();
  const [rejectOpen, setRejectOpen] = useState(false);
  const [rollbackOpen, setRollbackOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);

  const onApprove = async () => {
    setBusy(true);
    try {
      await api.approvePromotion(tenantId, promotionId, {
        group: "@payments-leads",
        bundle_hash_bound: bundleHash,
      });
      push({ title: "Approval recorded", description: "Sigstore keyless OIDC signature attested.", tone: "ok" });
      router.refresh();
    } catch (e) {
      push({ title: "Approve failed", description: (e as Error).message, tone: "alert" });
    } finally {
      setBusy(false);
    }
  };

  const onReject = async () => {
    setBusy(true);
    try {
      await api.rejectPromotion(tenantId, promotionId, reason);
      push({ title: "Promotion rejected", tone: "info" });
      router.refresh();
    } catch (e) {
      push({ title: "Reject failed", description: (e as Error).message, tone: "alert" });
    } finally {
      setBusy(false);
      setRejectOpen(false);
    }
  };

  const onRollback = async () => {
    setBusy(true);
    try {
      await api.rollbackPromotion(tenantId, promotionId, reason);
      push({ title: "Rollback initiated", tone: "warn" });
      router.refresh();
    } catch (e) {
      push({ title: "Rollback failed", description: (e as Error).message, tone: "alert" });
    } finally {
      setBusy(false);
      setRollbackOpen(false);
    }
  };

  if (status === "rolled_back" || status === "rejected" || status === "cancelled") {
    return (
      <Card>
        <CardContent className="p-4 text-xs text-muted-foreground">No actions available — terminal state.</CardContent>
      </Card>
    );
  }

  return (
    <>
      <Card id="approve">
        <CardHeader>
          <CardTitle>Approval</CardTitle>
          <CardDescription>
            Your signature is bound to the bundle's diff hash. If the bundle is amended after approval, prior approvals
            are invalidated.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          {status === "pending_approval" && (
            <>
              <Button onClick={onApprove} disabled={busy} className="w-full">
                Approve
              </Button>
              <Button variant="destructive" onClick={() => setRejectOpen(true)} disabled={busy} className="w-full">
                Reject
              </Button>
            </>
          )}
          {(status === "deploying" || status === "canary_dwell") && (
            <Button variant="destructive" onClick={() => setRollbackOpen(true)} disabled={busy} className="w-full">
              Rollback now
            </Button>
          )}
        </CardContent>
      </Card>

      <Dialog open={rejectOpen} onOpenChange={setRejectOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reject promotion</DialogTitle>
            <DialogDescription>
              Recorded in the attestation chain. Author is notified via webhook.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-1 p-4">
            <Label htmlFor="reject-reason">Reason</Label>
            <Textarea id="reject-reason" rows={4} value={reason} onChange={(e) => setReason(e.target.value)} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRejectOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" disabled={busy || !reason} onClick={onReject}>
              Reject
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={rollbackOpen} onOpenChange={setRollbackOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Manual rollback</DialogTitle>
            <DialogDescription>
              Forces the delivery adapter (Argo Rollouts or GrowthBook flag flip) to revert to the prior known-good
              state immediately. Auto-rollback would have fired on the next SLO-check failure regardless.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-1 p-4">
            <Label htmlFor="rb-reason">Reason</Label>
            <Textarea id="rb-reason" rows={4} value={reason} onChange={(e) => setReason(e.target.value)} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRollbackOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" disabled={busy || !reason} onClick={onRollback}>
              Rollback
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
