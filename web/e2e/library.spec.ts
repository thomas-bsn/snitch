import { expect, test } from "@playwright/test";

test("un membre filtre la bibliothèque et lit une plage d’un clip partagé ou privé", async ({
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
    name: "synthetic-shared.mp4",
    mimeType: "video/mp4",
    buffer: Buffer.from("synthetic-browser-video-content"),
  });
  await page.getByRole("button", { name: "Envoyer la vidéo" }).click();
  await expect(page.getByRole("status")).toContainText(
    "a été vérifiée et ajoutée",
  );
  await expect(
    page.getByText("synthetic-shared.mp4", { exact: true }),
  ).toBeVisible();
  await page.getByText("Détails et lecture", { exact: true }).click();
  const range = await page.evaluate(async () => {
    const video = document.querySelector("video")!;
    const response = await fetch(video.src, {
      headers: { Range: "bytes=2-6" },
    });
    return {
      status: response.status,
      range: response.headers.get("Content-Range"),
      bytes: await response.arrayBuffer().then((b) => b.byteLength),
    };
  });
  expect(range).toEqual({ status: 206, range: "bytes 2-6/31", bytes: 5 });

  await page.locator("#uploads form select").selectOption("private");
  await page.getByLabel("Fichier vidéo").setInputFiles({
    name: "synthetic-private.mp4",
    mimeType: "video/mp4",
    buffer: Buffer.from("synthetic-private-video-content"),
  });
  await page.getByRole("button", { name: "Envoyer la vidéo" }).click();
  await expect(
    page.getByText("synthetic-private.mp4", { exact: true }),
  ).toBeVisible();
  await page.locator("#library > form select").first().selectOption("shared");
  const filteredClipsResponse = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return (
      url.pathname === "/api/clips" &&
      url.searchParams.get("visibility") === "shared"
    );
  });
  await page
    .locator("#library > form")
    .evaluate((form) => (form as HTMLFormElement).requestSubmit());
  expect((await filteredClipsResponse).status()).toBe(200);
  await expect(
    page.getByText("synthetic-private.mp4", { exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByText("synthetic-shared.mp4", { exact: true }),
  ).toBeVisible();
});
