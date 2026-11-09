import { expect, request, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// Mise à jour et retour arrière du runtime d'un modèle prêt : l'écran Engines montre les modèles liés du Worker
// choisi. Le parcours complet (ordre, inférence de test, publication, retour arrière) est couvert par les tests
// d'intégration de l'API et du Worker ; le serveur E2E n'a ni clé de confiance ni modèle prêt.

const origin = () => process.env.SNITCH_TEST_URL!;

test("l'écran Engines explique les modèles liés et ne commande rien en le regardant", async ({
  browser,
}) => {
  await installAdministrator().catch(() => undefined);
  const api = await request.newContext({
    baseURL: origin(),
    extraHTTPHeaders: { Origin: origin() },
    storageState: await sessionFor(ADMIN),
  });
  const csrf = (await (await api.get("/api/auth/me")).json()).csrf_token;
  const name = `synthetic-bound-${Date.now()}`;
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
    await page.goto("/admin/voice?view=engines");
    await page.getByRole("searchbox", { name: "Search a Worker" }).fill(name);
    await page.getByRole("button", { name }).click();
    await expect(
      page.getByRole("heading", { name: "Bound models" }),
    ).toBeVisible();
    await expect(page.getByText("nothing is downloaded again")).toBeVisible();
    await expect(
      page.getByText("No ready model on an approved engine for this Worker."),
    ).toBeVisible();

    const bindings = await (
      await api.get("/api/admin/voice-engines/bindings")
    ).json();
    expect(
      bindings.bindings.filter(
        (b: { worker_id: string }) => b.worker_id === worker.id,
      ),
    ).toEqual([]);
  } finally {
    await api.delete(`/api/admin/workers/${worker.id}`, {
      headers: { "X-CSRF-Token": csrf },
    });
  }
});

// Plusieurs versions d'un engine à la demande (#376). Le serveur E2E n'a pas de modèle prêt : les liaisons et la
// vue des engines sont complétées par des réponses synthétiques, les ordres sont interceptés et leur corps vérifié.
// Le parcours complet côté serveur est couvert par TestFoundation/voice_model_runtime_on_demand.
test("un modèle d'une famille à la demande se met à jour par version, revient en arrière, et un engine en échec se remplace", async ({
  browser,
}) => {
  await installAdministrator().catch(() => undefined);
  const api = await request.newContext({
    baseURL: origin(),
    extraHTTPHeaders: { Origin: origin() },
    storageState: await sessionFor(ADMIN),
  });
  const csrf = (await (await api.get("/api/auth/me")).json()).csrf_token;
  const name = `synthetic-family-${Date.now()}`;
  const created = await api.post("/api/admin/workers", {
    headers: { "X-CSRF-Token": csrf },
    data: { name, max_concurrency: 1 },
  });
  const worker = (await created.json()) as { id: string };
  const digest = "a".repeat(64);
  const previous = "b".repeat(64);
  try {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.route("**/api/admin/voice-engines/bindings", (route) =>
      route.fulfill({
        json: {
          bindings: [
            {
              worker_id: worker.id,
              model_id: "synthetic-titanet",
              model_label: "Synthetic TitaNet",
              engine: "nemo-titanet",
              family: "nemo",
              versions: ["3.0.0", "3.1.0"],
              kind: "fingerprint",
              digest,
              version: "3.0.0",
              revoked: false,
              previous_digest: previous,
              previous_version: "2.9.0",
              previous_revoked: false,
              history: [],
            },
          ],
        },
      }),
    );
    await page.route("**/api/admin/voice-engines/workers", async (route) => {
      const response = await route.fetch();
      const view = await response.json();
      for (const row of view.workers)
        if (row.worker_id === worker.id)
          row.engines = [
            {
              digest: "c".repeat(64),
              engine: "nemo-toolkit",
              version: "9.9.9",
              kind: "fingerprint",
              source: "on_demand",
              state: "failed",
              cause_code: "package_not_found",
              bytes_received: 0,
              bytes_total: 0,
              order_id: "ord_synthetic",
              generation: 1,
              revoked: false,
              author_name: "synthetic",
              updated_at: new Date().toISOString(),
              history: [],
              cancel_requested: false,
              cleanup: "",
              order_intent: "install",
              order_status: "failed",
              order_cause_code: "package_not_found",
              waiting: null,
            },
          ];
      await route.fulfill({ response, json: view });
    });
    const runtimeBodies: Record<string, unknown>[] = [];
    await page.route("**/api/admin/voice-model-runtime-orders", (route) => {
      runtimeBodies.push(route.request().postDataJSON());
      return route.fulfill({
        status: 409,
        json: { error: "engine_order_in_progress" },
      });
    });
    const familyBodies: Record<string, unknown>[] = [];
    await page.route("**/api/admin/voice-engines/on-demand-orders", (route) => {
      familyBodies.push(route.request().postDataJSON());
      return route.fulfill({ json: { results: [] } });
    });

    await page.goto("/admin/voice?view=engines");
    await page.getByRole("searchbox", { name: "Search a Worker" }).fill(name);
    await page.getByRole("button", { name }).click();
    await expect(
      page.getByRole("heading", { name: "Bound models" }),
    ).toBeVisible();

    // The family model is listed; the picker offers the known versions of the family but the bound one.
    await expect(page.getByText("Synthetic TitaNet")).toBeVisible();
    await expect(page.locator("datalist option")).toHaveCount(1);
    await page.getByLabel("Target version for Synthetic TitaNet").fill("3.1.0");
    await page.getByRole("button", { name: "Update", exact: true }).click();
    await expect(page.getByText("the engine is installed first")).toBeVisible();
    expect(runtimeBodies).toEqual([]);
    await page.getByRole("button", { name: "Confirm the update" }).click();
    await expect.poll(() => runtimeBodies.length).toBe(1);
    expect(runtimeBodies[0]).toMatchObject({
      worker_id: worker.id,
      model_id: "synthetic-titanet",
      version: "3.1.0",
      intent: "update",
    });
    expect(runtimeBodies[0]).not.toHaveProperty("digest");

    // Rollback goes to the previous version already installed, by identity.
    await page.getByRole("button", { name: "Go back to 2.9.0" }).click();
    await page.getByRole("button", { name: "Go back to 2.9.0" }).last().click();
    await expect.poll(() => runtimeBodies.length).toBe(2);
    expect(runtimeBodies[1]).toMatchObject({
      intent: "rollback",
      digest: previous,
    });

    // A failed on-demand engine is replaced by another version of its family and task, not by a signed release.
    await page
      .getByRole("button", {
        name: "Replace nemo-toolkit 9.9.9 with another release",
      })
      .click();
    await expect(page.getByLabel("Engine family")).toBeDisabled();
    await page.getByLabel("Version", { exact: false }).last().fill("3.1.0");
    const install = page.getByRole("button", { name: /^Install/ });
    await expect(install).toBeEnabled();
    await install.click();
    await expect.poll(() => familyBodies.length).toBe(1);
    expect(familyBodies[0]).toMatchObject({
      family: "nemo",
      version: "3.1.0",
      kind: "fingerprint",
    });
  } finally {
    await api.delete(`/api/admin/workers/${worker.id}`, {
      headers: { "X-CSRF-Token": csrf },
    });
  }
});
