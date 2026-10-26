import { expect, test } from "@playwright/test";

test("admin prévisualise puis confirme un lot sans inclure les pistes ambiguës", async ({
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

  const upload = async (filename: string) => {
    await page.getByLabel("Fichier vidéo").setInputFiles({
      name: filename,
      mimeType: "video/mp4",
      buffer: Buffer.from(`synthetic-batch-${filename}`),
    });
    await page.getByRole("button", { name: "Envoyer la vidéo" }).click();
    await expect(page.getByRole("status")).toContainText(
      "a été vérifiée et ajoutée",
    );
  };

  await upload("batch-resolved.mp4");
  const resolvedCard = page
    .locator(".clip-card")
    .filter({ hasText: "batch-resolved.mp4" });
  await resolvedCard.getByText("Détails et lecture", { exact: true }).click();
  await resolvedCard.getByLabel("Piste audio").selectOption("2");
  await resolvedCard.getByRole("button", { name: "Transcrire" }).click();
  await expect(page.getByRole("status")).toContainText(
    "Transcription planifiée",
  );

  await upload("batch-ambiguous.mp4");
  const ids = await page.evaluate(async () => {
    const [clips, me] = await Promise.all([
      fetch("/api/clips?page_size=100").then((response) => response.json()),
      fetch("/api/auth/me").then((response) => response.json()),
    ]);
    const resolved = clips.items.find(
      (clip: { original_name: string }) =>
        clip.original_name === "batch-resolved.mp4",
    );
    const ambiguous = clips.items.find(
      (clip: { original_name: string }) =>
        clip.original_name === "batch-ambiguous.mp4",
    );
    if (!resolved || !ambiguous)
      throw new Error("synthetic batch clips were not visible");
    const jobs = await fetch(
      `/api/admin/jobs?clip_id=${resolved.id}&step=transcription.audio.2`,
      { credentials: "same-origin" },
    ).then((response) => response.json());
    for (const job of jobs.items) {
      if (["pending", "leased", "retry_wait"].includes(job.status)) {
        await fetch(`/api/admin/jobs/${job.id}`, {
          method: "DELETE",
          headers: { "X-CSRF-Token": me.csrf_token },
          credentials: "same-origin",
        });
      }
    }
    return { resolved: resolved.id, ambiguous: ambiguous.id };
  });

  const batch = page.getByLabel("Planification par lot");
  await batch.getByRole("button", { name: "Prévisualiser le lot" }).click();
  await expect(batch).toContainText("1 éligible(s)");
  await expect(batch).toContainText("1 piste(s) à sélectionner");
  page.once("dialog", (dialog) => dialog.accept());
  await batch.getByRole("button", { name: "Confirmer et planifier 1" }).click();
  await expect(
    page.getByText("1 tâche(s) ajoutée(s) à la file.", { exact: true }),
  ).toBeVisible();

  const jobCounts = await page.evaluate(async (clipIds) => {
    const jobsFor = async (clip_id: number) =>
      fetch(
        `/api/admin/jobs?clip_id=${clip_id}&step=transcription.audio.2&status=pending`,
        {
          credentials: "same-origin",
        },
      ).then((response) => response.json());
    return {
      resolved: await jobsFor(clipIds.resolved),
      ambiguous: await jobsFor(clipIds.ambiguous),
    };
  }, ids);
  expect(jobCounts.resolved.total).toBe(1);
  expect(jobCounts.ambiguous.total).toBe(0);
});
