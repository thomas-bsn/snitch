import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// Sortir d'une installation d'engine en échec (#351). Le serveur E2E n'a pas de clé de confiance : l'état d'échec
// est donc fourni par des réponses d'API simulées, et l'on vérifie ce que l'écran propose et ce qu'il envoie.
// Le comportement réel (annulation coopérative, reprise, nettoyage borné) est couvert par les tests
// d'intégration Go et les tests du Worker.

const DIGEST_A = "a".repeat(64);
const DIGEST_B = "b".repeat(64);
const manifest = (version: string) => ({
  schema: 1,
  engine: "nemo-titanet",
  version,
  adapter_version: 1,
  python: "3.12",
  platform: "linux-x86_64",
  cuda: "cpu",
  license: "Apache-2.0",
  kind: "fingerprint",
  architecture: "titanet",
  disk_bytes: 4096,
  vram_bytes: 0,
  wheels: [],
});
const entry = (over: Record<string, unknown>) => ({
  digest: DIGEST_A,
  engine: "nemo-titanet",
  version: "1.0.0",
  kind: "fingerprint",
  state: "failed",
  cause_code: "disk_full",
  bytes_received: 0,
  bytes_total: 0,
  cancel_requested: false,
  cleanup: "",
  order_id: "ord_synthetic_engine_order_0001",
  generation: 1,
  revoked: false,
  author_name: "admin",
  updated_at: new Date().toISOString(),
  history: [],
  ...over,
});

test("un échec propose Réessayer, Remplacer et Nettoyer, et un ordre en cours Annuler", async ({
  browser,
}) => {
  await installAdministrator().catch(() => undefined);
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  let engines = [
    entry({}),
    entry({
      digest: DIGEST_B,
      version: "2.0.0",
      state: "installing",
      cause_code: "",
      order_id: "ord_synthetic_engine_order_0002",
    }),
  ];
  const sent: string[] = [];
  await page.route("**/api/admin/voice-engines/workers", (route) =>
    route.fulfill({
      json: {
        workers: [
          {
            worker_id: "wk_synthetic_recovery",
            name: "synthetic-recovery",
            connected: true,
            engines,
          },
        ],
      },
    }),
  );
  await page.route("**/api/admin/voice-engines", (route) =>
    route.request().method() === "GET"
      ? route.fulfill({
          json: {
            engines: [
              { digest: DIGEST_A, manifest: manifest("1.0.0"), revoked: false },
              { digest: DIGEST_B, manifest: manifest("2.0.0"), revoked: false },
              {
                digest: "c".repeat(64),
                manifest: { ...manifest("3.0.0"), engine: "pyannote" },
                revoked: false,
              },
            ],
          },
        })
      : route.fallback(),
  );
  await page.route("**/api/admin/workers/*/voice-engines/*/*", (route) => {
    const verb = route.request().url().split("/").pop();
    sent.push(`${route.request().method()} ${verb}`);
    return verb === "retry"
      ? route.fulfill({ status: 201, json: {} })
      : route.fulfill({
          json: { status: verb === "cancel" ? "cancelling" : "requested" },
        });
  });

  await page.goto("/admin/voice?view=engines");
  await page.getByRole("button", { name: "synthetic-recovery" }).click();

  const failed = "nemo-titanet 1.0.0";
  const running = "nemo-titanet 2.0.0";
  // Failed: the three recovery actions, and no Cancel (nothing is running).
  await expect(
    page.getByRole("button", { name: `Retry the installation of ${failed}` }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", {
      name: `Replace ${failed} with another release`,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", {
      name: `Clean up the incomplete files of ${failed}`,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", {
      name: `Cancel the installation of ${failed}`,
    }),
  ).toHaveCount(0);
  // Running: Cancel only.
  await expect(
    page.getByRole("button", {
      name: `Cancel the installation of ${running}`,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: `Retry the installation of ${running}` }),
  ).toHaveCount(0);
  const { violations } = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
    .analyze();
  expect(violations).toEqual([]);

  await page
    .getByRole("button", { name: `Retry the installation of ${failed}` })
    .click();
  await page
    .getByRole("button", { name: `Cancel the installation of ${running}` })
    .click();
  await expect(
    page.getByText("Cancellation requested", { exact: false }),
  ).toBeVisible();
  engines = [
    entry({ cleanup: "requested" }),
    entry({
      digest: DIGEST_B,
      version: "2.0.0",
      state: "installing",
      cause_code: "",
      cancel_requested: true,
    }),
  ];
  await page
    .getByRole("button", {
      name: `Clean up the incomplete files of ${failed}`,
    })
    .click();
  await expect(
    page.getByText("Cleanup requested", { exact: false }).first(),
  ).toBeVisible();
  expect(sent).toEqual(["POST retry", "POST cancel", "POST cleanup"]);
  // After the request, the running install shows the pending cancellation instead of a second Cancel.
  await expect(
    page.getByRole("button", {
      name: `Cancel the installation of ${running}`,
    }),
  ).toHaveCount(0);

  // Replace: a failed release never blocks the choice, and only compatible releases of the same engine are listed.
  await page
    .getByRole("button", { name: `Replace ${failed} with another release` })
    .click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toContainText("A failed release never prevents");
  await expect(dialog).toContainText("nemo-titanet · 2.0.0");
  await expect(dialog).not.toContainText("pyannote");
  await expect(dialog).not.toContainText("nemo-titanet · 1.0.0");
});
