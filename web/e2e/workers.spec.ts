import { expect, test } from "@playwright/test";

test("un administrateur crée un worker puis révoque son jeton", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByLabel("Code d’installation")
    .fill(process.env.SNITCH_TEST_CODE!);
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-123");
  await page.getByRole("button", { name: "Créer l’administrateur" }).click();

  await page.getByLabel("Nom du worker").fill("synthetic-worker");
  await page.getByLabel("Concurrence maximale").fill("2");
  await page.getByRole("button", { name: "Créer un worker" }).click();
  await expect(
    page.getByRole("list", { name: "Workers" }).getByText("Absent"),
  ).toBeVisible();
  const token = page
    .locator('p[role="status"]')
    .filter({ hasText: "Jeton à copier maintenant" });
  await expect(token).toContainText("Jeton à copier maintenant: rw_");
  await expect(page.getByRole("list", { name: "Workers" })).toContainText(
    "synthetic-worker",
  );

  const worker = page
    .getByRole("list", { name: "Workers" })
    .getByRole("listitem")
    .filter({ hasText: "synthetic-worker" });
  await page.evaluate(async () => {
    const me = await fetch("/api/auth/me").then((response) => response.json());
    const created = await fetch("/api/admin/workers", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": me.csrf_token,
        Origin: location.origin,
      },
      body: JSON.stringify({ name: "pulse-fixture", max_concurrency: 1 }),
    });
    if (!created.ok)
      throw new Error("synthetic pulse worker registration failed");
    const { token } = await created.json();
    const pulse = await fetch("/api/worker/pulse", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ capacity: 1 }),
    });
    if (!pulse.ok) throw new Error("synthetic worker pulse failed");
  });
  await page.reload();
  await expect(page.getByRole("list", { name: "Workers" })).toBeVisible();
  await expect(
    page.getByRole("list", { name: "Workers" }).getByText("En ligne"),
  ).toBeVisible();
  await worker.getByRole("button", { name: "Révoquer le jeton" }).click();
  await expect(worker).toContainText("Révoqué");
  await expect(token).toHaveCount(0);
});
