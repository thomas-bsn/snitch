import { expect, test } from "@playwright/test";

test("admin diarise un clip transcrit et lit le Locuteur local de chaque segment", async ({
  page,
}) => {
  await page.goto("/legacy/");
  await page
    .getByLabel("Code d’installation")
    .fill(process.env.SNITCH_TEST_CODE!);
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-123");
  await page.getByRole("button", { name: "Créer l’administrateur" }).click();

  await page.getByLabel("Fichier vidéo").setInputFiles({
    name: "synthetic-voices.mp4",
    mimeType: "video/mp4",
    buffer: Buffer.from("synthetic-voices-video"),
  });
  await page.getByRole("button", { name: "Envoyer la vidéo" }).click();
  await expect(page.getByRole("status")).toContainText(
    "a été vérifiée et ajoutée",
  );
  await page.getByText("Détails et lecture", { exact: true }).click();
  await page.getByLabel("Piste audio").selectOption("2");

  await page.getByRole("button", { name: "Transcrire" }).click();
  await expect(page.getByRole("status")).toContainText(
    "Transcription planifiée",
  );

  // A simulated worker speaks only the worker HTTP API with its limited token.
  await page.evaluate(async () => {
    const me = await fetch("/api/auth/me").then((response) => response.json());
    const worker = await fetch("/api/admin/workers", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": me.csrf_token,
        Origin: location.origin,
      },
      body: JSON.stringify({ name: "synthetic-voice", max_concurrency: 2 }),
    }).then((response) => response.json());
    (window as unknown as { workerToken: string }).workerToken = worker.token;
    const call = (path: string, body?: unknown, lease?: string) =>
      fetch(path, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${worker.token}`,
          ...(lease ? { "X-Worker-Lease-Token": lease } : {}),
          ...(body ? { "Content-Type": "application/json" } : {}),
        },
        body: body ? JSON.stringify(body) : undefined,
      });
    await call("/api/worker/pulse", {
      capacity: 2,
      supported_steps: ["transcription.audio.*", "voice.*"],
    });
    const assignment = await call("/api/worker/jobs/claim").then((response) =>
      response.json(),
    );
    await call(
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
            {
              start: 2.5,
              end: 3.4,
              text: "Merci",
              words: [{ word: "Merci", start: 2.5, end: 3.0 }],
            },
          ],
        },
      },
      assignment.lease_token,
    );
  });

  // Access to a real model must be confirmed explicitly; tests use the fake provider.
  const pipeline = page.locator("#voice-pipeline");
  await pipeline.scrollIntoViewIfNeeded();
  await pipeline.getByLabel("Fournisseur de diarisation").selectOption("fake");
  await pipeline
    .getByLabel("Modèle de diarisation")
    .fill("fixture-diarization");
  await pipeline.getByLabel("Version du modèle").fill("fixture-v1");
  await pipeline
    .getByRole("button", { name: "Enregistrer et versionner" })
    .click();
  await expect(
    pipeline.getByText("Configuration vocale enregistrée."),
  ).toBeVisible();

  await page.getByRole("button", { name: "Diariser les voix" }).click();
  await expect(page.getByText("Traitement vocal planifié")).toBeVisible();
  await expect(page.getByText("Diarisation · en file")).toBeVisible();

  // The worker returns two local speakers and then their fingerprints.
  await page.evaluate(async () => {
    const token = (window as unknown as { workerToken: string }).workerToken;
    const call = (path: string, body?: unknown, lease?: string) =>
      fetch(path, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          ...(lease ? { "X-Worker-Lease-Token": lease } : {}),
          ...(body ? { "Content-Type": "application/json" } : {}),
        },
        body: body ? JSON.stringify(body) : undefined,
      });
    const claim = async () =>
      call("/api/worker/jobs/claim").then((response) => response.json());
    const diarization = await claim();
    const input = await fetch(`/api/worker/jobs/${diarization.job.id}/input`, {
      headers: {
        Authorization: `Bearer ${token}`,
        "X-Worker-Lease-Token": diarization.lease_token,
      },
    }).then((response) => response.json());
    const config = input.voice.config;
    await call(
      `/api/worker/jobs/${diarization.job.id}/complete`,
      {
        input_version: diarization.job.input_version,
        result: {
          provider: config.provider,
          model: config.model,
          model_version: config.model_version,
          audio_stream_index: 2,
          speakers: [
            { label: "SPEAKER_00", turns: [{ start: 0, end: 2.2 }] },
            { label: "SPEAKER_01", turns: [{ start: 2.2, end: 4.2 }] },
          ],
        },
      },
      diarization.lease_token,
    );
    const fingerprints = await claim();
    await call(
      `/api/worker/jobs/${fingerprints.job.id}/complete`,
      {
        input_version: fingerprints.job.input_version,
        result: {
          provider: config.provider,
          embedding_model: "fixture-embedding",
          embedding_model_version: "emb-v1",
          audio_stream_index: 2,
          diarization_job_id: diarization.job.id,
          dimensions: 2,
          fingerprints: [
            {
              speaker_label: "SPEAKER_00",
              scope: "speaker",
              start: 0,
              end: 2.2,
              quality: 0.9,
              embedding: [0.6, 0.8],
            },
            {
              speaker_label: "SPEAKER_01",
              scope: "speaker",
              start: 2.2,
              end: 2.5,
              quality: 0.9,
              embedding: [0.8, 0.6],
            },
          ],
        },
      },
      fingerprints.lease_token,
    );
  });

  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  const voices = page.getByRole("region", { name: "Voix du clip" });
  await expect(voices.getByRole("heading")).toContainText("Terminé");
  const speakers = voices.getByRole("list", { name: "Locuteurs locaux" });
  await expect(speakers).toContainText("SPEAKER_00 · non identifié");
  await expect(speakers).toContainText("1 empreinte(s), 1 utilisable(s)");
  await expect(speakers).toContainText("1 empreinte(s), 0 utilisable(s)");
  const transcript = page.getByRole("region", {
    name: "Transcription horodatée",
  });
  await expect(
    transcript.getByText("SPEAKER_00 · non identifié"),
  ).toBeVisible();
  await expect(
    transcript.getByText("SPEAKER_01 · non identifié"),
  ).toBeVisible();
});
