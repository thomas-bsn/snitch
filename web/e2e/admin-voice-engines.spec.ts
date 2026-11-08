import AxeBuilder from "@axe-core/playwright";
import { expect, request, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

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
