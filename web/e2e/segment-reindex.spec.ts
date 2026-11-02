import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("un segment corrigé reste à réindexer si le fournisseur est indisponible", async ({
  page,
}) => {
  const clipId = await prepareDiarizedClip(page);

  await page.goto("/admin/semantic");
  await page
    .getByLabel("Embeddings service address")
    .fill(process.env.SNITCH_TEST_EMBEDDINGS_URL!);
  await page.getByLabel("Embeddings model").fill("synthetic-reindex-v1");
  await page.getByRole("button", { name: "Test", exact: true }).click();
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByRole("status")).toContainText("saved");

  // Saving a profile queues the current transcript. Wait for the real index before
  // correcting it, so this journey exercises reindexing rather than initial indexing.
  await expect
    .poll(async () => {
      const response = await page.request.get(
        `${process.env.SNITCH_TEST_URL}/api/admin/embeddings`,
      );
      const status = await response.json();
      return status.indexed_segments;
    })
    .toBe(3);

  const control = await page.request.post(
    `${process.env.SNITCH_TEST_EMBEDDINGS_URL}/control`,
    { data: { down: true } },
  );
  expect(control.status()).toBe(204);

  await page.goto(`/clips/${clipId}`);
  const transcript = page.getByRole("list", { name: "Transcript segments" });
  const firstSegment = transcript.getByRole("listitem").first();
  await firstSegment.getByRole("button", { name: "Edit text" }).click();
  await firstSegment
    .getByLabel("Segment text")
    .fill("Texte corrigé synthétique");
  await firstSegment.getByRole("button", { name: "Save text" }).click();
  await expect(firstSegment).toContainText("Texte corrigé synthétique");

  await expect
    .poll(async () => {
      const response = await page.request.get(
        `${process.env.SNITCH_TEST_URL}/api/admin/embeddings`,
      );
      const status = await response.json();
      return status.reindex_pending;
    })
    .toBe(1);

  await page.goto("/admin/semantic");
  await expect(
    page.getByText(/corrected segments awaiting reindexing: 1/),
  ).toBeVisible();
});
