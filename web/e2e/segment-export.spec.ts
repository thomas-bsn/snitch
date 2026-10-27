import { readFile } from "node:fs/promises";
import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("un administrateur télécharge l’export des segments corrigés", async ({
  page,
}) => {
  await prepareDiarizedClip(page);
  const seeded = await page.evaluate(async () => {
    const me = await fetch("/api/auth/me").then((response) => response.json());
    const clips = await fetch("/api/clips").then((response) => response.json());
    const clipId = clips.items[0].id;
    const response = await fetch(`/api/clips/${clipId}/segments/0/text`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": me.csrf_token,
        Origin: location.origin,
      },
      body: JSON.stringify({
        text: "Bien joué, Alice !",
        expected_version: 0,
      }),
    });
    return response.status;
  });
  expect(seeded).toBe(200);

  await page.goto("/admin/exports");
  const card = page.getByRole("heading", { name: "Exports" }).locator("..");
  await expect(card).toContainText(
    "Download the currently corrected segments in NDJSON format.",
  );
  const downloading = page.waitForEvent("download");
  await card.getByRole("link", { name: "Download corrected segments" }).click();
  const download = await downloading;
  expect(download.suggestedFilename()).toBe("segment-corrections.ndjson");
  const content = await readFile((await download.path())!, "utf8");
  expect(content).not.toBe("");
  const lines = content
    .trim()
    .split("\n")
    .map((line) => JSON.parse(line));
  expect(lines).toHaveLength(1);
  expect(lines[0]).toMatchObject({
    index: 0,
    corrected_text: "Bien joué, Alice !",
    author: { username: "administrator" },
    version: 1,
  });
  expect(lines[0].original_text).not.toBe("Bien joué, Alice !");
});
