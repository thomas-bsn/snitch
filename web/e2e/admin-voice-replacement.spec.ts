import { expect, request, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// Remplacement d'un modèle de diarisation sur un Worker (#308) : le nouveau est installé et testé pendant que
// l'ancien reste en service, Snitch décide la bascule puis ordonne le retrait de l'ancien. Données synthétiques ;
// le Worker est simulé par l'API.

const OLD = "pyannote-community-1";
const NEW = "nvidia-sortformer-4spk-v2.1";

test("un administrateur remplace le modèle de diarisation d'un Worker, de l'échec au retrait de l'ancien", async ({
  browser,
}) => {
  await installAdministrator();
  const session = await sessionFor(ADMIN);
  const origin = process.env.SNITCH_TEST_URL!;
  const admin = await request.newContext({
    baseURL: origin,
    extraHTTPHeaders: { Origin: origin },
    storageState: session,
  });
  const me = await (await admin.get("/api/auth/me")).json();
  const csrf = { "X-CSRF-Token": me.csrf_token };
  const { token, id } = await (
    await admin.post("/api/admin/workers", {
      headers: csrf,
      data: { name: "worker-remplacement", max_concurrency: 1 },
    })
  ).json();
  const bearer = { Authorization: `Bearer ${token}` };
  const pulled = async () =>
    (
      await (
        await admin.get("/api/worker/voice-models", { headers: bearer })
      ).json()
    ).orders as {
      order_id: string;
      action: string;
      model_id: string;
      revision: string;
    }[];
  const report = async (
    order: { order_id: string; model_id: string; revision: string },
    state: object,
  ) =>
    expect(
      (
        await admin.post("/api/worker/voice-models/status", {
          headers: bearer,
          data: {
            order_id: order.order_id,
            model_id: order.model_id,
            revision: order.revision,
            ...state,
          },
        })
      ).status(),
    ).toBe(204);
  const ack = async (orderId: string) =>
    expect(
      (
        await admin.post(`/api/worker/voice-model-orders/${orderId}/ack`, {
          headers: bearer,
          data: { accepted: true },
        })
      ).status(),
    ).toBe(204);
  // The Worker already holds the old model, ready (a report without order, as before the orders).
  expect(
    (
      await admin.post("/api/worker/voice-models/status", {
        headers: bearer,
        data: { model_id: OLD, state: "ready" },
      })
    ).status(),
  ).toBe(204);

  const context = await browser.newContext({ storageState: session });
  const page = await context.newPage();
  await page.goto("/admin/voice");
  const section = page.getByRole("region", { name: "Models per Worker" });
  const cell = section
    .getByRole("row", { name: /worker-remplacement/ })
    .locator("td")
    .nth(1);
  await expect(cell).toContainText("Ready");

  // Replace: an explicit confirmation that says the old model stays until the switch.
  await cell.getByRole("button", { name: "Change" }).click();
  const dialog = page.getByRole("alertdialog");
  await expect(dialog).toContainText("stays active");
  await dialog.getByRole("button", { name: "Order the installation" }).click();
  await expect(cell).toContainText("Requested");
  await expect(cell).toContainText("it stays active until the switch");

  // The disk cannot hold both: the failure is shown, the old model is kept, and Retry or Cancel is offered.
  const [install] = await pulled();
  expect(install.action).toBe("install");
  expect(install.model_id).toBe(NEW);
  await ack(install.order_id);
  await report(install, { state: "failed", cause_code: "disk_full" });
  await page.reload();
  await expect(cell).toContainText("Failed");
  await cell.getByRole("button", { name: "Cancel the replacement" }).click();
  await expect(cell).toContainText("Ready");
  await expect(cell.getByRole("button", { name: "Change" })).toBeVisible();

  // Second attempt: ready, then the removal of the old model is an order of its own.
  await cell.getByRole("button", { name: "Change" }).click();
  await dialog.getByRole("button", { name: "Order the installation" }).click();
  await expect(cell).toContainText("Requested");
  const [again] = await pulled();
  await ack(again.order_id);
  await report(again, { state: "verifying" });
  await report(again, { state: "ready", occupied_bytes: 471_403_401 });
  const [removal] = await pulled();
  expect(removal.action).toBe("remove");
  expect(removal.model_id).toBe(OLD);
  await page.reload();
  await expect(cell).toContainText("Removing");
  await ack(removal.order_id);
  await report(removal, { state: "removed" });
  await page.reload();
  await expect(cell).toContainText("Ready");
  await expect(cell).toContainText("nvidia/diar_streaming_sortformer");
  await expect(cell.getByText("Replacing")).toHaveCount(0);

  await admin.delete(`/api/admin/workers/${id}`, { headers: csrf });
  await context.close();
  await admin.dispose();
});
