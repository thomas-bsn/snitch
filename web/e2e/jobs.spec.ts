import { expect, test } from "@playwright/test";

test("un administrateur planifie et filtre une tâche persistante", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByLabel("Code d’installation")
    .fill(process.env.RECEIPTS_TEST_CODE!);
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-123");
  await page.getByRole("button", { name: "Créer l’administrateur" }).click();

  await page.getByLabel("Fichier vidéo").setInputFiles({
    name: "synthetic-job-clip.mp4",
    mimeType: "video/mp4",
    buffer: Buffer.from("synthetic-job-video-content"),
  });
  await page.getByRole("button", { name: "Envoyer la vidéo" }).click();
  await expect(page.getByRole("status")).toContainText(
    "a été vérifiée et ajoutée",
  );
  const clipID = await page.evaluate(async () => {
    const response = await fetch("/api/clips?page=1&page_size=10");
    const page = await response.json();
    return page.items[0].id as number;
  });

  await page.getByLabel("ID du clip", { exact: true }).fill(String(clipID));
  await page.getByLabel("Étape", { exact: true }).fill("fixture.inspect");
  await page.getByLabel("Version d’entrée").fill("fixture-v1");
  await page.getByLabel("Priorité").fill("5");
  await page.getByRole("button", { name: "Planifier l’étape" }).click();
  await expect(page.getByRole("status")).toHaveText("Tâche planifiée.");
  await expect(
    page.getByRole("heading", { name: "File de tâches (1)" }),
  ).toBeVisible();
  await expect(
    page
      .getByRole("list", { name: "Tâches" })
      .getByText(new RegExp(`fixture\\.inspect.*clip ${clipID} · pending`)),
  ).toBeVisible();

  await page.getByLabel("ID du clip", { exact: true }).fill(String(clipID));
  await page.getByLabel("Étape", { exact: true }).fill("fixture.inspect");
  await page.getByLabel("Version d’entrée").fill("fixture-v1");
  await page.getByRole("button", { name: "Planifier l’étape" }).click();
  await expect(page.getByRole("status")).toHaveText(
    "Cette étape et cette version sont déjà dans la file.",
  );
  await expect(
    page.getByRole("heading", { name: "File de tâches (1)" }),
  ).toBeVisible();

  await page.getByLabel("Étape à filtrer").fill("fixture.inspect");
  await page.getByRole("button", { name: "Filtrer les tâches" }).click();
  await expect(
    page.getByRole("list", { name: "Tâches" }).getByText("fixture.inspect"),
  ).toBeVisible();

  const workerToken = await page.evaluate(async () => {
    const me = await fetch("/api/auth/me").then((response) => response.json());
    const response = await fetch("/api/admin/workers", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": me.csrf_token,
        Origin: location.origin,
      },
      body: JSON.stringify({
        name: "synthetic-retry-worker",
        max_concurrency: 1,
      }),
    });
    if (!response.ok) throw new Error("synthetic worker registration failed");
    return (await response.json()).token as string;
  });
  const workerRequest = async (path: string, body?: unknown) =>
    page.evaluate(
      async ({ path, body, token }) => {
        const response = await fetch(path, {
          method: "POST",
          headers: {
            Authorization: `Bearer ${token}`,
            ...(body ? { "Content-Type": "application/json" } : {}),
          },
          body: body ? JSON.stringify(body) : undefined,
        });
        if (!response.ok)
          throw new Error(`synthetic worker request failed: ${path}`);
        return response.json();
      },
      { path, body, token: workerToken },
    );
  await workerRequest("/api/worker/pulse", { capacity: 1 });
  const lease = await workerRequest("/api/worker/jobs/claim");
  await workerRequest(`/api/worker/jobs/${lease.job.id}/fail`, {
    error: "invalid_input",
  });
  await page.getByLabel("État de la tâche").selectOption("dead");
  await page.getByRole("button", { name: "Filtrer les tâches" }).click();
  await expect(page.getByRole("list", { name: "Tâches" })).toContainText(
    "dead",
  );
  await page.getByRole("button", { name: "Relancer" }).click();
  await expect(page.getByRole("status")).toHaveText(
    `Tâche ${lease.job.id} relancée.`,
  );
  await page.getByLabel("État de la tâche").selectOption("");
  await page.getByRole("button", { name: "Filtrer les tâches" }).click();
  await page.getByRole("button", { name: "Historique" }).click();
  const history = page.getByRole("list", {
    name: `Historique de la tâche ${lease.job.id}`,
  });
  await expect(history).toContainText("marked_dead");
  await expect(history).toContainText("admin_relaunched");
  await page.getByRole("button", { name: "Annuler" }).click();
  await expect(page.getByRole("status")).toHaveText(
    `Tâche ${lease.job.id} annulée.`,
  );
});
