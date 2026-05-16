import { expect, test } from "@playwright/test";

// Golden-path E2E: navigate from overview → tasks → plan-approval → execution stream.
// This test runs against the dev server with mocked backend payloads in /lib/mocks.

test.describe("plan approval flow", () => {
  test("overview surfaces the trust narrative", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Overview" })).toBeVisible();
    await expect(page.getByText("evidence — not vibes")).toBeVisible();
  });

  test("a senior engineer can review a plan, see the cost preview, and approve it", async ({ page }) => {
    await page.goto("/tasks/task_01HZAB_x4/approve");
    await expect(page.getByRole("heading", { name: /Add idempotency key/ })).toBeVisible();
    await expect(page.getByText("cost est.")).toBeVisible();
    await expect(page.getByText("$0.42")).toBeVisible();
    await expect(page.getByText(/Webhook signature verification path/)).toBeVisible();
    await expect(page.getByText("budget cap")).toBeVisible();
    // Approve button is present and labelled as the primary action.
    await expect(page.getByRole("button", { name: "Approve plan" })).toBeVisible();
  });

  test("attestation viewer shows predicate body + verify button", async ({ page }) => {
    await page.goto("/attestations/rekor:b2cdd9f4c8a1a3e2");
    await expect(page.getByRole("heading", { name: "Attestation" })).toBeVisible();
    await expect(page.getByText("predicate body")).toBeVisible();
    await expect(page.getByRole("button", { name: /Verify end-to-end/ })).toBeVisible();
  });

  test("promotions page lists pending approvals", async ({ page }) => {
    await page.goto("/promotions");
    await expect(page.getByRole("tab", { name: /Approval inbox/ })).toBeVisible();
  });

  test("memory page lists active conventions with confidence", async ({ page }) => {
    await page.goto("/memory");
    await expect(page.getByText(/API errors return/)).toBeVisible();
  });
});
