import { readFileSync } from "node:fs";
import AxeBuilder from "@axe-core/playwright";
import { expect, request, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";
import {
  planOrders,
  type EngineWorker,
  type Release,
} from "../src/app/admin/voice/engineOrders";

// L'onglet Engines d'Administration → Voix, Worker d'abord : rien n'est installé ni commandé tant que
// l'administrateur ne confirme pas. Le parcours complet ordre → installation → synchronisation est couvert par
// les tests d'intégration de l'API et du Worker : aucune clé de confiance n'est configurée sur le serveur E2E.

const origin = () => process.env.SNITCH_TEST_URL!;

test("l'onglet Engines liste les Workers et n'ordonne rien sans confirmation", async ({
  browser,
}) => {
  await installAdministrator().catch(() => undefined);
  const api = await request.newContext({
    baseURL: origin(),
    extraHTTPHeaders: { Origin: origin() },
    storageState: await sessionFor(ADMIN),
  });
  const csrf = (await (await api.get("/api/auth/me")).json()).csrf_token;
  const name = `synthetic-engines-${Date.now()}`;
  const created = await api.post("/api/admin/workers", {
    headers: { "X-CSRF-Token": csrf },
    data: { name, max_concurrency: 1 },
  });
  const worker = (await created.json()) as { id: string };
  try {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    const manifest = JSON.parse(
      readFileSync(
        "../worker/engine-releases/speechbrain-ecapa-1.0.3-r1/manifest.json",
        "utf8",
      ),
    );
    await page.route("**/api/admin/voice-engines", (route) =>
      route.fulfill({
        json: {
          engines: [{ digest: "a".repeat(64), revoked: false, manifest }],
        },
      }),
    );
    await page.goto("/admin/voice?view=engines");
    await expect(page.getByRole("tab", { name: "Engines" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    await page.getByRole("searchbox", { name: "Search a Worker" }).fill(name);
    await page.getByRole("button", { name }).click();
    await expect(
      page.getByText("No engine has been ordered or installed on this Worker."),
    ).toBeVisible();
    await expect(page.getByRole("tab", { name: /Installed/ })).toBeVisible();
    await expect(page.getByRole("tab", { name: /To handle/ })).toBeVisible();
    await expect(page.getByRole("tab", { name: /In progress/ })).toBeVisible();

    await page.getByRole("button", { name: "Add an engine" }).click();
    const dialog = page.getByRole("dialog", {
      name: `Add an engine to ${name}`,
    });
    await expect(dialog).toContainText("Nothing starts before you confirm");
    // Without a chosen release the plan is empty and the install button stays disabled.
    await expect(dialog).toContainText("0 order(s)");
    await expect(
      dialog.getByRole("button", { name: "Install 0 engine(s)" }),
    ).toBeDisabled();
    // A scope with no checked Worker must never silently target the current Worker.
    await dialog.getByRole("checkbox").check();
    await expect(dialog).toContainText("1 order(s)");
    await dialog.getByRole("combobox").selectOption("checked");
    await expect(dialog).toContainText("0 order(s)");
    await expect(
      dialog.getByRole("button", { name: "Install 0 engine(s)" }),
    ).toBeDisabled();
    await dialog.getByRole("combobox").selectOption("this");
    await expect(dialog).toContainText("1 order(s)");
    const { violations } = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
      .analyze();
    expect(violations).toEqual([]);
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).toBeHidden();

    // Looking at the screen ordered nothing: the Worker still has no engine.
    const view = await (
      await api.get("/api/admin/voice-engines/workers")
    ).json();
    const row = view.workers.find(
      (w: { worker_id: string }) => w.worker_id === worker.id,
    );
    expect(row.engines).toEqual([]);
  } finally {
    await api.delete(`/api/admin/workers/${worker.id}`, {
      headers: { "X-CSRF-Token": csrf },
    });
  }
});

test("le plan sérialise les releases d’un même engine par Worker", () => {
  const manifest = JSON.parse(
    readFileSync(
      "../worker/engine-releases/speechbrain-ecapa-1.0.3-r1/manifest.json",
      "utf8",
    ),
  );
  const worker: EngineWorker = {
    worker_id: "synthetic",
    name: "Synthetic",
    connected: false,
    engines: [],
    disk_reserved_bytes: 0,
  };
  const releases: Release[] = [
    { digest: "a".repeat(64), revoked: false, manifest },
    {
      digest: "b".repeat(64),
      revoked: false,
      manifest: { ...manifest, version: "next" },
    },
  ];
  const plan = planOrders([worker], releases);
  expect(plan.accepted).toHaveLength(1);
  expect(plan.items[1].refusal).toBe("engine_order_in_progress");
  expect(plan.disk).toBe(manifest.disk_bytes);
});
