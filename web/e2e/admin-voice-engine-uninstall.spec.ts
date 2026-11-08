import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// Désinstaller un engine : la fenêtre montre les modèles liés, jobs engagés et transitions d'empreintes avant
// tout ordre. Aucune clé de confiance n'est configurée sur le serveur E2E : les réponses de l'API des engines sont
// simulées ici ; le parcours réel (refus, drainage, reprise, retrait) est couvert par les tests d'intégration.

const digest = "a".repeat(64);
const entry = (state: string, extra: Record<string, unknown> = {}) => ({
  digest,
  engine: "nemo-titanet",
  version: "1.0.0",
  kind: "fingerprint",
  state,
  cause_code: "",
  bytes_received: 0,
  bytes_total: 0,
  order_id: "ord_synthetic_000000000001",
  generation: 1,
  revoked: false,
  author_name: "synthetic",
  updated_at: new Date().toISOString(),
  order_intent: "install",
  order_status: "done",
  order_cause_code: "",
  waiting: null,
  history: [],
  ...extra,
});
const worker = (engines: unknown[]) => ({
  workers: [
    {
      worker_id: "wk_synthetic_uninstall",
      name: "synthetic-uninstall",
      connected: true,
      engines,
    },
  ],
});

test("Désinstaller explique ce qui bloque, puis ordonne le retrait de cette seule version", async ({
  browser,
}) => {
  await installAdministrator().catch(() => undefined);
  const page = await (
    await browser.newContext({ storageState: await sessionFor(ADMIN) })
  ).newPage();
  let removable = false;
  let current = worker([entry("ready")]);
  const orders: unknown[] = [];
  await page.route("**/api/admin/voice-engines/workers", (route) =>
    route.fulfill({ json: current }),
  );
  await page.route("**/api/admin/voice-engines/uninstall-preview*", (route) =>
    route.fulfill({
      json: {
        state: "ready",
        removable,
        blockers: {
          open_order: false,
          models: removable
            ? []
            : [
                {
                  model_id: "synthetic-model",
                  label: "Synthetic fingerprint",
                  kind: "fingerprint",
                  state: "ready",
                  busy: true,
                  in_transition: true,
                },
              ],
        },
      },
    }),
  );
  await page.route("**/api/admin/voice-engine-orders", async (route) => {
    orders.push(route.request().postDataJSON());
    current = worker([entry("removing", { order_intent: "uninstall" })]);
    await route.fulfill({
      json: {
        results: [{ worker_id: "wk_synthetic_uninstall", status: "created" }],
      },
    });
  });

  await page.goto("/admin/voice?view=engines");
  await page.getByRole("button", { name: "synthetic-uninstall" }).click();
  await page.getByRole("button", { name: "Uninstall" }).click();
  const dialog = page.getByRole("dialog", {
    name: "Uninstall nemo-titanet 1.0.0 from synthetic-uninstall",
  });
  await expect(dialog).toContainText("Synthetic fingerprint");
  await expect(dialog).toContainText("job running");
  await expect(dialog).toContainText("fingerprint transition open");
  await expect(
    dialog.getByRole("button", { name: "Uninstall" }),
  ).toBeDisabled();
  const { violations } = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
    .analyze();
  expect(violations).toEqual([]);
  expect(orders).toEqual([]);
  await dialog.getByRole("button", { name: "Cancel" }).click();

  removable = true;
  await page.getByRole("button", { name: "Uninstall" }).click();
  await expect(dialog).toContainText("Nothing uses this version");
  await dialog.getByRole("button", { name: "Uninstall" }).click();
  await expect(page.getByText("Removing")).toBeVisible();
  expect(orders).toHaveLength(1);
  expect(orders[0]).toMatchObject({ digest, intent: "uninstall" });
});
