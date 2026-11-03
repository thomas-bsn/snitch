import { expect, type Page } from "@playwright/test";
import { transcribeAndDiarize, uploadClip } from "./library-fixture";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// Installs an administrator and prepares a synthetic, diarized clip through the HTTP fixture.
export async function prepareDiarizedClip(
  page: Page,
  options: { secondFingerprintEnd?: number; peaks?: boolean } = {},
): Promise<number> {
  await installAdministrator();
  const clipId = await uploadClip("synthetic-voices.mp4", "shared");
  await transcribeAndDiarize(clipId, options);
  await page.context().addCookies((await sessionFor(ADMIN)).cookies);
  await page.goto(`/clips/${clipId}`);
  await expect(
    page.getByRole("heading", { name: "synthetic-voices.mp4" }),
  ).toBeVisible();
  return clipId;
}
