"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { PageHeader } from "@/components/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input, Textarea } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { useToast } from "@/components/ui/toaster";

const schema = z.object({
  description: z.string().min(8, "describe in at least a sentence"),
  repo: z.string().min(3),
  base_sha: z.string().optional(),
});
type FormT = z.infer<typeof schema>;

export default function NewTaskPage() {
  const router = useRouter();
  const { push } = useToast();
  const [busy, setBusy] = useState(false);
  const form = useForm<FormT>({
    resolver: zodResolver(schema),
    defaultValues: { repo: "github.com/acme/payments" },
  });

  const onSubmit = async (v: FormT) => {
    setBusy(true);
    try {
      // In production: api.submitTask(...). The server-side handler builds
      // the plan and redirects to /tasks/[id]/approve once it lands.
      await new Promise((r) => setTimeout(r, 600));
      const fakeId = `task_${Math.random().toString(36).slice(2, 10)}`;
      push({ title: "Task submitted", description: "Planner is building the preview.", tone: "ok" });
      router.push(`/tasks/${fakeId}/approve`);
    } catch (e) {
      push({ title: "Submission failed", description: (e as Error).message, tone: "alert" });
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <PageHeader title="New task" description="Crucible plans before it acts. You will review the plan before the agent writes a single line." />
      <Card className="max-w-2xl">
        <CardHeader>
          <CardTitle>Describe the work</CardTitle>
          <CardDescription>
            Be specific. Mention the entity, the path, the desired behaviour. Don't paste a prompt — paste a ticket.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <div>
              <Label htmlFor="description">Task description</Label>
              <Textarea
                id="description"
                rows={6}
                placeholder="e.g. add idempotency-key support to /webhooks/stripe/refund using the existing idempotency_keys table; keep the signature-verification path on the entry of the handler"
                {...form.register("description")}
              />
              {form.formState.errors.description && (
                <p className="mt-1 text-xs text-accent-alert">{form.formState.errors.description.message}</p>
              )}
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <Label htmlFor="repo">Repository</Label>
                <Input id="repo" {...form.register("repo")} />
              </div>
              <div>
                <Label htmlFor="base_sha">Base SHA (optional)</Label>
                <Input id="base_sha" placeholder="HEAD" {...form.register("base_sha")} />
              </div>
            </div>
            <Button type="submit" disabled={busy}>
              {busy ? "Submitting…" : "Submit task"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </>
  );
}
