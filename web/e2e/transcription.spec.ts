import { expect, test } from "@playwright/test";

test("admin choisit une piste globale, planifie puis ouvre la vidéo au timestamp transcrit", async ({
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

  const vocabulary = page.locator("#vocabulary");
  await vocabulary.getByLabel("Terme").fill("Final Fantasy XIV");
  await vocabulary
    .getByLabel("Variantes, séparées par des virgules")
    .fill("FFXIV, Final Fantasy 14");
  await vocabulary.getByLabel("Graphie préférée").fill("FFXIV");
  await vocabulary.getByRole("button", { name: "Ajouter le terme" }).click();

  await page.getByLabel("Fichier vidéo").setInputFiles({
    name: "synthetic-multitrack.mp4",
    mimeType: "video/mp4",
    buffer: Buffer.from("synthetic-multitrack-video"),
  });
  await page.getByRole("button", { name: "Envoyer la vidéo" }).click();
  await expect(page.getByRole("status")).toContainText(
    "a été vérifiée et ajoutée",
  );
  await page.getByText("Détails et lecture", { exact: true }).click();
  await expect(
    page.getByText(
      "Plusieurs pistes audio sont possibles. Sélectionnez celle à transcrire.",
    ),
  ).toBeVisible();
  await expect(page.getByLabel("Piste audio")).toHaveValue("");
  await page.getByLabel("Piste audio").selectOption("2");
  await page.getByRole("button", { name: "Transcrire" }).click();
  await expect(page.getByRole("status")).toContainText(
    "Transcription planifiée",
  );

  const result = await page.evaluate(async () => {
    const me = await fetch("/api/auth/me").then((response) => response.json());
    const worker = await fetch("/api/admin/workers", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": me.csrf_token,
        Origin: location.origin,
      },
      body: JSON.stringify({
        name: "synthetic-transcriber",
        max_concurrency: 1,
      }),
    }).then((response) => response.json());
    const workerCall = (path: string, body?: unknown, lease?: string) =>
      fetch(path, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${worker.token}`,
          ...(lease ? { "X-Worker-Lease-Token": lease } : {}),
          ...(body ? { "Content-Type": "application/json" } : {}),
        },
        body: body ? JSON.stringify(body) : undefined,
      });
    await workerCall("/api/worker/pulse", {
      capacity: 1,
      supported_steps: ["transcription.audio.*"],
    });
    const assignment = await workerCall("/api/worker/jobs/claim").then(
      (response) => response.json(),
    );
    const input = await fetch(`/api/worker/jobs/${assignment.job.id}/input`, {
      headers: {
        Authorization: `Bearer ${worker.token}`,
        "X-Worker-Lease-Token": assignment.lease_token,
      },
    }).then((response) => response.json());
    const completion = await workerCall(
      `/api/worker/jobs/${assignment.job.id}/complete`,
      {
        input_version: assignment.job.input_version,
        result: {
          provider: "fixture",
          model: "deterministic-v1",
          version: "deterministic-v1",
          language: "fr",
          audio_stream_index: 2,
          segments: [
            {
              start: 1.25,
              end: 2.1,
              text: "Bien joué !",
              words: [
                { word: "Bien", start: 1.25, end: 1.52 },
                { word: "joué", start: 1.55, end: 1.9 },
              ],
            },
          ],
        },
      },
      assignment.lease_token,
    );
    return {
      step: assignment.job.step,
      status: completion.status,
      vocabulary: input.vocabulary,
    };
  });
  expect(result).toMatchObject({
    step: "transcription.audio.2",
    status: 200,
    vocabulary: [
      {
        term: "Final Fantasy XIV",
        variants: ["FFXIV", "Final Fantasy 14"],
        preferred_spelling: "FFXIV",
      },
    ],
  });

  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  await expect(page.getByText("Bien", { exact: true })).toBeVisible();
  await expect(page.getByText("joué", { exact: true })).toBeVisible();
  await page.evaluate(() => {
    const video = document.querySelector("video")!;
    Object.defineProperty(video, "currentTime", {
      configurable: true,
      set: (value) => {
        (window as Window & { transcriptTime?: number }).transcriptTime = value;
      },
    });
    video.play = () => Promise.resolve();
  });
  await page.getByRole("button", { name: "00:01" }).click();
  await expect
    .poll(() =>
      page.evaluate(
        () => (window as Window & { transcriptTime?: number }).transcriptTime,
      ),
    )
    .toBe(1.25);
});
