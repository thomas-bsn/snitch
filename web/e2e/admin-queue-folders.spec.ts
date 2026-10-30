import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

test("l'administrateur pilote la file et voit les dossiers disponibles", async ({
  browser,
}) => {
  await installAdministrator();
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/admin/queue");
  await page
    .locator("summary")
    .filter({ hasText: "Concurrency settings" })
    .click();
  await page.getByLabel("Global concurrency").fill("3");
  await page.getByLabel("Pause new acquisitions").check();
  const firstSave = page.waitForResponse(
    (r) =>
      r.url().endsWith("/api/admin/job-controls") &&
      r.request().method() === "PUT" &&
      r.ok(),
  );
  await page.getByRole("button", { name: "Save controls" }).click();
  await firstSave;
  await page.reload();
  await page
    .locator("summary")
    .filter({ hasText: "Concurrency settings" })
    .click();
  await expect(page.getByLabel("Global concurrency")).toHaveValue("3");
  await expect(page.getByLabel("Pause new acquisitions")).toBeChecked();
  expect(
    (
      await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
        .analyze()
    ).violations,
  ).toEqual([]);
  await page.getByLabel("Pause new acquisitions").uncheck();
  const saved = page.waitForResponse(
    (r) =>
      r.url().endsWith("/api/admin/job-controls") &&
      r.request().method() === "PUT" &&
      r.ok(),
  );
  await page.getByRole("button", { name: "Save controls" }).click();
  await saved;
  await page.goto("/import");
  await page.getByLabel("choose files").setInputFiles({
    name: "synthetic.mp4",
    mimeType: "video/mp4",
    buffer: Buffer.alloc(20000, 1),
  });
  await expect(
    page.getByRole("listitem").filter({ hasText: "synthetic.mp4" }),
  ).toContainText("Added to the library", { timeout: 30000 });
  const me = await (await page.request.get("/api/auth/me")).json();
  const headers = {
    Origin: process.env.SNITCH_TEST_URL!,
    "X-CSRF-Token": me.csrf_token,
  };
  const clips = await (await page.request.get("/api/clips")).json();
  await page.goto("/admin/queue");
  await page.locator("summary").filter({ hasText: "Schedule task" }).click();
  await page
    .getByLabel("Clip ID", { exact: true })
    .fill(String(clips.items[0].id));
  await page.getByLabel("Step", { exact: true }).fill("fixture.inspect");
  await page.getByLabel("Input version").fill("fixture-v1");
  await page.getByLabel("Maximum attempts").fill("1");
  const scheduled = page.waitForResponse(
    (r) =>
      r.url().endsWith("/api/admin/jobs") &&
      r.request().method() === "POST" &&
      r.status() === 201,
  );
  await page.getByRole("button", { name: "Schedule task" }).click();
  const job = await (await scheduled).json();
  const workerResponse = await page.request.post("/api/admin/workers", {
    headers,
    data: { name: "fixture-worker", max_concurrency: 1 },
  });
  expect(workerResponse.status()).toBe(201);
  const worker = await workerResponse.json();
  const authorization = { Authorization: `Bearer ${worker.token}` };
  expect(
    (
      await page.request.post("/api/worker/pulse", {
        headers: authorization,
        data: { capacity: 1, supported_steps: ["fixture.*"] },
      })
    ).ok(),
  ).toBe(true);
  const claim = await (
    await page.request.post("/api/worker/jobs/claim", {
      headers: authorization,
    })
  ).json();
  expect(
    (
      await page.request.post(`/api/worker/jobs/${claim.job.id}/fail`, {
        headers: {
          ...authorization,
          "X-Worker-Lease-Token": claim.lease_token,
        },
        data: { error: "invalid_input" },
      })
    ).ok(),
  ).toBe(true);
  await page.goto("/admin/queue");
  const item = page
    .getByRole("listitem")
    .filter({ hasText: "fixture.inspect" });
  await expect(item).toContainText("Exhausted");
  await item.getByRole("button", { name: "Retry" }).click();
  await expect(item).toContainText("Pending");
  await item.getByRole("button", { name: "History" }).click();
  await expect(item).toContainText("Relaunched by administrator");
  await item.getByRole("button", { name: "Cancel" }).click();
  await expect(item).toContainText("Cancelled");
  expect(job.id).toBe(claim.job.id);
  await page.getByLabel("Filter by step").fill("different.step");
  await page.getByRole("button", { name: "Filter tasks" }).click();
  await expect(page.getByText("No tasks.")).toBeVisible();
  await page.goto("/admin/folders");
  await expect(
    page.getByLabel("Root folder accessible to Snitch"),
  ).toBeVisible();
  await expect(page.getByText("No watched folders configured.")).toBeVisible();
  expect(
    (
      await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
        .analyze()
    ).violations,
  ).toEqual([]);
  await context.close();
});
