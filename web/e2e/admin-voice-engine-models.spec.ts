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
