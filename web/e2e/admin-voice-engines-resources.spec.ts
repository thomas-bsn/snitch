import { expect, request, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// Les contraintes matérielles d'un Worker (Python, plateforme, CUDA, RAM, VRAM, disque) sont visibles dans
// l'onglet Engines. Le refus exact d'une release et les réservations de disque concurrentes sont couverts par
// les tests d'intégration de l'API : aucune clé de confiance n'est configurée sur le serveur E2E.

const origin = () => process.env.SNITCH_TEST_URL!;

test("l'onglet Engines expose le matériel déclaré par le Worker", async ({
  browser,
}) => {
  await installAdministrator().catch(() => undefined);
  const api = await request.newContext({
    baseURL: origin(),
    extraHTTPHeaders: { Origin: origin() },
    storageState: await sessionFor(ADMIN),
  });
  const csrf = (await (await api.get("/api/auth/me")).json()).csrf_token;
  const name = `synthetic-hardware-${Date.now()}`;
  const created = await api.post("/api/admin/workers", {
    headers: { "X-CSRF-Token": csrf },
    data: { name, max_concurrency: 1 },
  });
  const worker = (await created.json()) as { id: string; token: string };
  try {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/admin/voice?view=engines");
    await page.getByRole("searchbox", { name: "Search a Worker" }).fill(name);
    await page.getByRole("button", { name }).click();
    // Un Worker qui n'a rien déclaré n'est pas bloqué : il vérifie lui-même avant d'installer.
    await expect(
      page.getByText("This Worker has not reported its hardware"),
    ).toBeVisible();

    const report = await request.newContext({
      baseURL: origin(),
      extraHTTPHeaders: { Authorization: `Bearer ${worker.token}` },
    });
    const posted = await report.post("/api/worker/voice-engines/hardware", {
      data: {
        python: "3.12",
        platform: "linux-x86_64",
        cuda: "",
        ram_bytes: 16 * 1024 ** 3,
        vram_bytes: 0,
        disk_free_bytes: 200 * 1024 ** 3,
        cpu_fallback: false,
      },
    });
    expect(posted.status()).toBe(204);
    await page.reload();
    await page.getByRole("searchbox", { name: "Search a Worker" }).fill(name);
    await page.getByRole("button", { name }).click();
    const machine = page.getByLabel("Machine");
    await expect(machine).toContainText("Python 3.12");
    await expect(machine).toContainText("no CUDA");
    await expect(machine).toContainText("RAM 16.0 GiB");
    await expect(machine).toContainText("free disk 200.0 GiB");
  } finally {
    await api.delete(`/api/admin/workers/${worker.id}`, {
      headers: { "X-CSRF-Token": csrf },
    });
  }
});
