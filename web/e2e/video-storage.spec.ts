import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// Single video root: configure it without an automatic scan, preview a requested
// scan, create a subfolder, upload into it (with overwrite), then change the root.
// Fixtures are synthetic; no real media is ever scanned.

const ROOT = process.env.SNITCH_TEST_VIDEO_ROOT;
const ROOT_2 = process.env.SNITCH_TEST_VIDEO_ROOT_2;

const video = (name: string, fill: number, size = 64 * 1024) => ({
  name,
  mimeType: "video/mp4",
  buffer: Buffer.alloc(size, fill),
});

const PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
  "base64",
);

const row = (page: Page, name: string) =>
  page.getByRole("listitem").filter({ hasText: name });

async function expectNoViolations(page: Page) {
  const { violations } = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
    .analyze();
  expect(
    violations.map(
      (v) => `${v.id}: ${v.nodes.map((n) => n.target.join(" ")).join(" | ")}`,
    ),
  ).toEqual([]);
}

test.describe.serial("racine vidéo et photo de profil", () => {
  test.skip(!ROOT || !ROOT_2, "requires the video-storage harness");

  test.beforeAll(installAdministrator);
  test.beforeEach(async ({ context }) => {
    await context.addCookies((await sessionFor(ADMIN)).cookies);
  });

  test("la racine se configure sans scan automatique", async ({ page }) => {
    await page.goto("/admin/folders");
    await page.getByLabel("Root folder accessible to Snitch").fill(ROOT!);
    await page.getByRole("button", { name: "Save folder" }).click();
    await expect(
      page.getByText("Folder saved. No scan started."),
    ).toBeVisible();
    await expect(
      page.getByText("No Worker is connected", { exact: false }),
    ).toBeVisible();
    // Nothing was discovered: the fixture only appears in the browser.
    await expect(page.getByText("browser-fixture.mp4")).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Requested files" }),
    ).toHaveCount(0);

    await page.getByLabel("New subfolder").fill("matches");
    await page.getByRole("button", { name: "Create folder" }).click();
    await expect(page.getByText("Folder created.")).toBeVisible();
    await expect(page.getByRole("button", { name: "matches" })).toBeVisible();
    await expectNoViolations(page);
  });

  test("une sélection est prévisualisée puis attend un Worker", async ({
    page,
  }) => {
    await page.goto("/library");
    await page
      .getByRole("button", { name: "Scan videos and manage folders" })
      .click();
    await page
      .getByRole("checkbox", { name: "Select browser-fixture.mp4" })
      .check();
    await page
      .getByRole("button", { name: "Prepare selection scan (1)" })
      .click();
    await expect(
      page.getByRole("heading", { name: "Confirm files to scan (1)" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Add to queue" }).click();
    await expect(page.getByText("Files added to the queue.")).toBeVisible();
    // Ready on disk, but still waiting for a Worker.
    await expect(
      page.getByText("Waiting for a Worker", { exact: true }),
    ).toBeVisible();
    await expectNoViolations(page);
  });

  test("un envoi choisit un sous-dossier et écrase après confirmation", async ({
    page,
  }) => {
    await page.goto("/import");
    await page.getByRole("button", { name: "Choose a folder" }).click();
    await page.getByRole("button", { name: "matches", exact: true }).click();
    await page.getByRole("button", { name: "Use this folder" }).click();
    await expect(page.getByText("/matches")).toBeVisible();

    await page.getByLabel("choose files").setInputFiles(video("game.mp4", 9));
    await expect(row(page, "game.mp4")).toContainText("Added to the library", {
      timeout: 30000,
    });
    await expect
      .poll(async () => {
        const data = await (
          await page.request.get("/api/admin/video-root/browse?path=matches")
        ).json();
        return data.items.some(
          (item: { path: string }) => item.path === "matches/game.mp4",
        );
      })
      .toBe(true);

    // Replacing the same name warns first, then installs the new content.
    page.once("dialog", (dialog) => dialog.accept());
    await page
      .getByLabel("choose files")
      .setInputFiles(video("game.mp4", 10, 48 * 1024));
    await expect
      .poll(
        async () => {
          const data = await (
            await page.request.get("/api/admin/video-root/browse?path=matches")
          ).json();
          const file = data.items.find(
            (item: { path: string }) => item.path === "matches/game.mp4",
          );
          return file?.size_bytes;
        },
        { timeout: 30000 },
      )
      .toBe(48 * 1024);
    await expectNoViolations(page);
  });

  test("changer la racine détache les fichiers sans perdre leurs résultats", async ({
    page,
  }) => {
    await page.goto("/admin/folders");
    const rootInput = page.getByLabel("Root folder accessible to Snitch");
    // This page starts with an empty path draft; the configured path is shown
    // in the root summary and only needs to be replaced in this form.
    page.once("dialog", (dialog) => dialog.accept());
    await rootInput.fill(ROOT_2!);
    await page.getByRole("button", { name: "Save folder" }).click();
    await expect(
      page.getByText("Root changed.", { exact: false }),
    ).toBeVisible();
    await expect(
      page.getByText("Missing", { exact: true }).first(),
    ).toBeVisible();
    await expect(
      page.getByText("No folders or video files here."),
    ).toBeVisible();
  });

  test("la photo de profil peut être ajoutée puis retirée", async ({
    page,
  }) => {
    await page.goto("/account");
    await page.getByLabel("Choose a photo (JPEG or PNG)").setInputFiles({
      name: "avatar.png",
      mimeType: "image/png",
      buffer: PNG,
    });
    await expect(page.getByText("Profile photo updated.")).toBeVisible();
    const avatars = page.locator('img[src^="/api/account/avatar"]');
    await expect(avatars.first()).toBeVisible();
    await page.getByRole("button", { name: "Remove photo" }).click();
    await expect(page.getByText("Profile photo updated.")).toBeVisible();
    await expect(avatars).toHaveCount(0);
    await expectNoViolations(page);
  });
});
