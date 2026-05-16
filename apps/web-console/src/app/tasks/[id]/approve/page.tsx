import { PlanApprovalFlow } from "@/components/plan-approval/plan-approval-flow";
import { PageHeader } from "@/components/page-header";
import { mockTask } from "@/lib/mocks";
import type { Task } from "@/lib/api";

export default async function ApprovePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  // The control plane fetch happens server-side with the tenant JWT in
  // production. The mock factory keeps the page legible without a backend.
  const task: Task = mockTask(id);
  return (
    <>
      <PageHeader
        title={task.description}
        description="Pre-execution review. Read the plan, set the budget, and sign — or amend and resubmit."
      />
      <PlanApprovalFlow task={task} />
    </>
  );
}
