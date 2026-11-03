import AxeBuilder from "@axe-core/playwright";
import { expect, request, test, type Page } from "@playwright/test";
import {
  ADMIN,
  MEMBER,
  createApprovedMember,
  installAdministrator,
  sessionFor,
} from "./shell-fixture";

// « Déjà sur le serveur » (#224) and the folder requests (#225, ADR 0033) over a synthetic Racine
// vidéo: top.mp4, league/2026/{match,other}.mp4, league/2026/sub/inner.mp4, secret/plan.mp4 (private).
// Nothing real is read. Cancelling a request by its requester is not in the spec of the lot D (ADR
// 0033 knows only approve and refuse), so it is not tested.

const ROOT = process.env.SNITCH_TEST_EXPLORER_ROOT;

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

const serverSection = (page: Page) =>
  page.getByRole("region", { name: "Already on the server" });

test.describe.serial("import depuis le serveur et demandes de dossier", () => {
  test.skip(!ROOT, "requires the explorer harness");

  test.beforeAll(async () => {
    await installAdministrator();
    const api = await request.newContext({
      baseURL: process.env.SNITCH_TEST_URL!,
      extraHTTPHeaders: { Origin: process.env.SNITCH_TEST_URL! },
      storageState: await sessionFor(ADMIN),
    });
    const me = await (await api.get("/api/auth/me")).json();
    const headers = { "X-CSRF-Token": me.csrf_token };
    const configured = await api.put("/api/admin/video-root", {
      headers,
      data: { path: ROOT },
    });
    expect(configured.status()).toBe(200);
    const sources = await (await api.get("/api/admin/video-root")).json();
    const privacy = await api.put("/api/admin/library/folders/privacy", {
      headers,
      data: { source_id: sources.root.id, folder: "secret", private: true },
    });
    expect(privacy.status()).toBe(200);
    await api.dispose();
    await createApprovedMember();
  });

  test("un membre voit le dossier privé verrouillé et ne peut pas le cocher", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await context.newPage();
    await page.goto("/import");
    const section = serverSection(page);
    await expect(section.getByText("secret")).toBeVisible();
    await expect(section.getByLabel("Select secret")).toBeDisabled();
    await expect(section.getByText("plan.mp4")).toHaveCount(0);
    // Creating a folder belongs to the upload destination, not to this section.
    await expect(section.getByText("New folder")).toHaveCount(0);
    await expectNoViolations(page);
    await context.close();
  });

  test("fichiers et dossier récursif : total compté, puis confirmation", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/import");
    const section = serverSection(page);
    await expect(section.getByText("New folder")).toHaveCount(0);

    await section.getByLabel("Select top.mp4").check();
    await section.getByLabel("Select league").check();
    await expect(section.getByText("2 item(s) selected")).toBeVisible();
    await section.getByRole("button", { name: "Count the videos" }).click();
    // top.mp4 plus the three videos of league, sub-folders included.
    await expect(
      section.getByText("4 video(s) will be added to the library."),
    ).toBeVisible();
    await expectNoViolations(page);

    // Cancelling the confirmation queues nothing; counting again then confirming does.
    await section.getByRole("button", { name: "Cancel" }).click();
    await expect(section.getByText(/will be added/)).toHaveCount(0);
    await section.getByRole("button", { name: "Count the videos" }).click();
    await section.getByRole("button", { name: "Add 4 video(s)" }).click();
    await expect(section.getByRole("status")).toContainText(
      "4 video(s) added.",
    );
    await expect(section.getByText("0 item(s) selected")).toBeVisible();
    await context.close();
  });

  test("un membre demande un dossier, l'envoi y est refusé, l'administrateur décide", async ({
    browser,
  }) => {
    const member = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await member.newPage();
    await page.goto("/import");
    await page.getByRole("button", { name: /Destination folder/ }).click();
    await page.getByLabel("Folder name").fill("requested-dir");
    await page.getByRole("button", { name: "New folder" }).click();
    await expect(
      page.getByText(/Request sent\. An administrator will decide/),
    ).toBeVisible();

    // Nothing may be uploaded into a folder that waits for a decision.
    const memberApi = await request.newContext({
      baseURL: process.env.SNITCH_TEST_URL!,
      extraHTTPHeaders: { Origin: process.env.SNITCH_TEST_URL! },
      storageState: await sessionFor(MEMBER),
    });
    const me = await (await memberApi.get("/api/auth/me")).json();
    const refused = await memberApi.post("/api/uploads", {
      headers: { "X-CSRF-Token": me.csrf_token },
      data: {
        filename: "x.mp4",
        size_bytes: 10,
        visibility: "shared",
        destination: "requested-dir",
      },
    });
    expect(refused.status()).toBe(409);
    await memberApi.dispose();

    const admin = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const adminPage = await admin.newPage();
    await adminPage.goto("/admin/users/signups");
    await expect(adminPage.getByText("/requested-dir »")).toBeVisible();
    await adminPage
      .getByRole("button", { name: "Approve the folder /requested-dir" })
      .click();
    await expect(
      adminPage.getByText("No folder request awaiting"),
    ).toBeVisible();

    // Once approved, the folder exists and the member can use it as a destination.
    await page.reload();
    await page.getByRole("button", { name: /Destination folder/ }).click();
    await expect(
      page.getByRole("button", { name: /requested-dir/ }),
    ).toBeVisible();
    await member.close();
    await admin.close();
  });

  test("un administrateur crée un dossier directement, sans demande", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/import");
    await page.getByRole("button", { name: /Destination folder/ }).click();
    await page.getByLabel("Folder name").fill("direct-dir");
    await page.getByRole("button", { name: "New folder" }).click();
    // The created folder becomes the destination at once.
    await expect(
      page.getByRole("button", { name: /Destination folder.*\/direct-dir/ }),
    ).toBeVisible();
    await expect(page.getByText(/Request sent/)).toHaveCount(0);
    await context.close();
  });

  test("une demande refusée ne crée rien", async ({ browser }) => {
    const member = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await member.newPage();
    await page.goto("/import");
    await page.getByRole("button", { name: /Destination folder/ }).click();
    await page.getByLabel("Folder name").fill("refused-dir");
    await page.getByRole("button", { name: "New folder" }).click();
    await expect(page.getByText(/Request sent/)).toBeVisible();

    const admin = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const adminPage = await admin.newPage();
    await adminPage.goto("/admin/users/signups");
    await adminPage
      .getByRole("button", { name: "Refuse the folder /refused-dir" })
      .click();
    await expect(
      adminPage.getByText("No folder request awaiting"),
    ).toBeVisible();

    await page.reload();
    await page.getByRole("button", { name: /Destination folder/ }).click();
    await expect(page.getByRole("button", { name: /refused-dir/ })).toHaveCount(
      0,
    );
    await member.close();
    await admin.close();
  });
});
