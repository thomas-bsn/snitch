import { expect, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// Previous interface, kept under /legacy/. The administrator is installed and signed in
// through the API (the installation wizard replaced the old form). This file is not named
// workers.spec.ts: Playwright filters by substring, so it would also run admin-workers.spec.ts.
test("un administrateur crée un worker puis révoque son jeton", async ({
  page,
  context,
}) => {
  await installAdministrator();
  await context.addCookies((await sessionFor(ADMIN)).cookies);
  await page.goto("/legacy/");

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

  const files = page.getByRole("region", { name: "Fichiers du worker" });
  const env = files.getByLabel("Contenu de .env");
  const origin = new URL(page.url()).origin;
  await expect(env).toHaveValue(new RegExp(`SNITCH_API_URL=${origin}\\n`));
  await expect(env).toHaveValue(/SNITCH_WORKER_TOKEN=rw_[A-Za-z0-9_-]{32,}\n/);
  await expect(env).toHaveValue(/TRANSCRIPTION_API_URL=A_RENSEIGNER\n/);
  await expect(env).not.toHaveValue(/\nTRANSCRIPTION_MODEL=/);
  await expect(env).toHaveValue(/HF_TOKEN=A_RENSEIGNER\n/);
  await expect(env).toHaveValue(/SNITCH_VOICE_DOWNLOAD_ON_START=1\n/);
  await expect(env).toHaveValue(/SNITCH_VOICE_CONFIRM_TERMS=1\n/);
  await expect(files.getByLabel("Contenu de compose.yaml")).toHaveValue(
    /pull_policy: always/,
  );
  await expect(files.getByLabel("Contenu de compose.yaml")).not.toHaveValue(
    /rw_/,
  );
  // The test server answers in plain HTTP, which the worker refuses.
  await expect(files.getByRole("alert")).toContainText("n’est pas en HTTPS");
  const download = page.waitForEvent("download");
  await files.getByRole("button", { name: "Télécharger .env" }).click();
  expect((await download).suggestedFilename()).toBe("snitch-worker.env");
  await expect(
    files.getByRole("button", { name: "Copier compose.yaml" }),
  ).toBeVisible();

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
      body: JSON.stringify({
        capacity: 1,
        models: {
          transcription: {
            provider: "openai-compatible",
            model: "large-v3",
            endpoint: "host.docker.internal:8000",
          },
          diarization: {
            provider: "pyannote",
            model: "pyannote/speaker-diarization-community-1",
            version: "4.0.1",
          },
        },
      }),
    });
    if (!pulse.ok) throw new Error("synthetic worker pulse failed");
  });
  await page.reload();
  await expect(page.getByRole("list", { name: "Workers" })).toBeVisible();
  await expect(
    page.getByRole("list", { name: "Workers" }).getByText("En ligne"),
  ).toBeVisible();
  const declared = page
    .getByRole("list", { name: "Workers" })
    .getByRole("listitem")
    .filter({ hasText: "pulse-fixture" });
  await expect(declared).toContainText(
    "Transcription : openai-compatible · large-v3 · host.docker.internal:8000, Diarisation : pyannote · pyannote/speaker-diarization-community-1 · 4.0.1, Empreinte vocale : non déclaré",
  );
  await expect(worker).toContainText("Modèles : non déclaré");
  await worker.getByRole("button", { name: "Révoquer le jeton" }).click();
  await expect(worker).toContainText("Révoqué");
  await expect(token).toHaveCount(0);
  await expect(files).toHaveCount(0);
});
