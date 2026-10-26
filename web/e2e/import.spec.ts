import AxeBuilder from "@axe-core/playwright";
import { test, expect, type Page, type Route } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// Import screen of the new interface: drop or choose videos, progress per file, resume
// after an interruption (even after a reload), cancel, and translated limit errors.
// The server of this suite cuts files in parts of 16 KiB and refuses more than 1 MiB.

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];
const video = (name: string, fill: number, size = 64 * 1024) => ({
  name,
  mimeType: "video/mp4",
  buffer: Buffer.alloc(size, fill),
});

async function expectNoViolations(page: Page) {
  const { violations } = await new AxeBuilder({ page })
    .withTags(TAGS)
    .analyze();
  expect(
    violations.map(
      (v) => `${v.id}: ${v.nodes.map((n) => n.target.join(" ")).join(" | ")}`,
    ),
  ).toEqual([]);
}

const row = (page: Page, name: string) =>
  page.getByRole("listitem").filter({ hasText: name });

/** Refuses the second part of a file once, as a cut connection would. */
async function cutAfterFirstPart(page: Page) {
  let parts = 0;
  const handler = async (route: Route) => {
    parts += 1;
    if (parts === 2) await route.abort("connectionreset");
    else await route.continue();
  };
  await page.route("**/api/uploads/*/chunks", handler);
  return () => page.unroute("**/api/uploads/*/chunks", handler);
}

test.describe.serial("import dans la nouvelle interface", () => {
  test.beforeAll(installAdministrator);
  test.beforeEach(async ({ context }) => {
    await context.addCookies((await sessionFor(ADMIN)).cookies);
  });

  test("des fichiers choisis sont envoyés avec une progression par fichier", async ({
    page,
  }) => {
    await page.goto("/import");
    await expect(page.getByRole("heading", { name: "Import" })).toBeVisible();
    await expectNoViolations(page);
    await page
      .getByLabel("choose files")
      .setInputFiles([video("first-pick.mp4", 1), video("second-pick.mp4", 2)]);
    for (const name of ["first-pick.mp4", "second-pick.mp4"]) {
      await expect(row(page, name)).toContainText("Added to the library", {
        timeout: 30000,
      });
      await expect(row(page, name).getByRole("progressbar")).toHaveAttribute(
        "aria-valuenow",
        "100",
      );
    }
    await expectNoViolations(page);
    await page.getByRole("link", { name: "Open the library" }).click();
    await expect(page.getByText("first-pick.mp4").first()).toBeVisible();
  });

  test("un glisser-déposer lance le transfert", async ({ page }) => {
    await page.goto("/import");
    await expect(page.getByText("Drop videos here")).toBeVisible();
    await page.evaluate(() => {
      const data = new DataTransfer();
      data.items.add(
        new File([new Uint8Array(20000).fill(7)], "dragged.mp4", {
          type: "video/mp4",
        }),
      );
      document
        .querySelector("input[type=file]")!
        .parentElement!.dispatchEvent(
          new DragEvent("drop", { dataTransfer: data, bubbles: true }),
        );
    });
    await expect(row(page, "dragged.mp4")).toContainText(
      "Added to the library",
      { timeout: 30000 },
    );
  });

  test("un transfert interrompu est repris", async ({ page }) => {
    await page.goto("/import");
    const restore = await cutAfterFirstPart(page);
    await page.getByLabel("choose files").setInputFiles(video("cut.mp4", 3));
    const item = row(page, "cut.mp4");
    await expect(item).toContainText("Interrupted");
    await expect(item.getByRole("alert")).toHaveText(
      "The connection was interrupted.",
    );
    await expectNoViolations(page);
    await restore();
    await item
      .getByRole("button", { name: "Resume the transfer of cut.mp4" })
      .click();
    await expect(item).toContainText("Added to the library", {
      timeout: 30000,
    });
  });

  test("après un rechargement, l'envoi est repris avec le même fichier", async ({
    page,
  }) => {
    await page.goto("/import");
    await cutAfterFirstPart(page);
    await page.getByLabel("choose files").setInputFiles(video("reload.mp4", 4));
    await expect(row(page, "reload.mp4")).toContainText("Interrupted");
    await page.unrouteAll({ behavior: "ignoreErrors" });
    await page.reload();
    const item = row(page, "reload.mp4");
    await expect(item).toContainText("Interrupted");
    await expectNoViolations(page);
    // Another file (other size) is refused, the same one is accepted.
    const picker = item.locator("input[type=file]");
    await picker.setInputFiles(video("reload.mp4", 4, 1000));
    await expect(item.getByRole("alert")).toContainText("not the same file");
    await picker.setInputFiles(video("reload.mp4", 4));
    await expect(item).toContainText("Added to the library", {
      timeout: 30000,
    });
  });

  test("un transfert interrompu peut être annulé", async ({ page }) => {
    await page.goto("/import");
    await cutAfterFirstPart(page);
    await page
      .getByLabel("choose files")
      .setInputFiles(video("abandoned.mp4", 5));
    const item = row(page, "abandoned.mp4");
    await expect(item).toContainText("Interrupted");
    await item
      .getByRole("button", { name: "Cancel the transfer of abandoned.mp4" })
      .click();
    await expect(item).toHaveCount(0);
    await page.unrouteAll({ behavior: "ignoreErrors" });
    await page.reload();
    await expect(page.getByRole("heading", { name: "Import" })).toBeVisible();
    await expect(row(page, "abandoned.mp4")).toHaveCount(0);
  });

  test("les limites et les formats refusés sont expliqués dans la langue choisie", async ({
    page,
  }) => {
    await page.goto("/import");
    await page.getByLabel("choose files").setInputFiles({
      name: "notes.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("not a video"),
    });
    await expect(page.getByRole("alert")).toHaveText(
      "“notes.txt” is not a video file.",
    );
    await page
      .getByLabel("choose files")
      .setInputFiles(video("huge.mp4", 6, 1024 * 1024 + 1));
    await expect(row(page, "huge.mp4").getByRole("alert")).toHaveText(
      "This file is too large.",
    );
    await expectNoViolations(page);

    await page.evaluate(() => localStorage.setItem("snitch-ui-language", "fr"));
    await page.goto("/import");
    await expect(page.getByRole("heading", { name: "Importer" })).toBeVisible();
    await page
      .getByLabel("choisissez des fichiers")
      .setInputFiles(video("enorme.mp4", 7, 1024 * 1024 + 1));
    await expect(row(page, "enorme.mp4").getByRole("alert")).toHaveText(
      "Ce fichier est trop volumineux.",
    );
    await page.evaluate(() => localStorage.removeItem("snitch-ui-language"));
  });

  test("écran d'import au téléphone", async ({ browser }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
      viewport: { width: 390, height: 780 },
    });
    const page = await context.newPage();
    await page.goto("/import");
    await expect(page.getByRole("heading", { name: "Import" })).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    await expectNoViolations(page);
    await context.close();
  });
});
