import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

test("un administrateur ajoute et configure un dossier synthétique", async ({
  browser,
}) => {
  await installAdministrator();
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/admin/folders");
  await page.getByLabel("Folder name").fill("Synthetic clips");
  await page
    .getByLabel("Folder path")
    .fill(process.env.SNITCH_TEST_SOURCE_ROOT!);
  await page.getByRole("button", { name: "Add folder" }).click();
  await expect(
    page.getByText("Synthetic clips", { exact: true }),
  ).toBeVisible();
  await page.getByText("Audio track rule").click();
  await page.getByLabel("Exact track title").fill("Voix");
  await page.getByLabel("Fallback audio stream index").fill("2");
  await page.getByRole("button", { name: "Save rule" }).click();
  await expect(page.getByRole("status").last()).toContainText("Saved");
  await page.getByRole("button", { name: "Scan", exact: true }).click();
  await expect(page.getByText(/Last scan:/)).toBeVisible();
  await expect
    .poll(async () => {
      const sources = await (
        await page.request.get("/api/admin/sources")
      ).json();
      const files = await (
        await page.request.get(
          `/api/admin/sources/${sources.items[0].id}/files`,
        )
      ).json();
      return files.items.some(
        (file: { status: string }) => file.status === "ready",
      );
    })
    .toBe(true);
  await page.getByText("Microphone track", { exact: true }).click();
  await expect(page.getByText("No microphone track designated.")).toBeVisible();
  await page.getByLabel("Detected track").selectOption("1");
  await page.getByLabel("Match by title").check();
  await page.getByLabel("New person").fill("Alice synthétique");
  await page.getByRole("button", { name: "Save rule" }).last().click();
  await expect(page.getByText("Assigned to Alice synthétique.")).toBeVisible();
  await page.getByRole("button", { name: "Remove designation" }).click();
  await expect(page.getByText("No microphone track designated.")).toBeVisible();

  expect(
    (
      await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
        .analyze()
    ).violations,
  ).toEqual([]);
  await context.close();
});
