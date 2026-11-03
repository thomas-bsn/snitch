import AxeBuilder from "@axe-core/playwright";
import { expect, request, test, type Page } from "@playwright/test";
import {
  SEGMENTS,
  correctSpeaker,
  correctText,
  nameVoice,
  transcribeAndDiarize,
  uploadClip,
} from "./library-fixture";
import {
  ADMIN,
  MEMBER,
  createApprovedMember,
  installAdministrator,
  sessionFor,
} from "./shell-fixture";

// Journeys of the library and of the page of a clip, in the new interface, on a synthetic
// library built through the API (a few bytes per video, a simulated worker).

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];

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

const noOverflow = (page: Page) =>
  page.evaluate(
    () => document.documentElement.scrollWidth <= window.innerWidth,
  );

test.describe.serial("bibliothèque et page d'un clip", () => {
  let sharedId = 0;
  let privateId = 0;

  test.beforeAll(installAdministrator);

  test("une bibliothèque vide invite à importer", async ({ browser }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/library");
    await expect(
      page.getByRole("heading", { level: 1, name: "Library" }),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "No clips yet" }),
    ).toBeVisible();
    await page.getByRole("link", { name: "Import videos" }).click();
    await expect(page).toHaveURL(/\/import$/);
    await context.close();
  });

  test.describe("avec des clips", () => {
    test.beforeAll(async () => {
      sharedId = await uploadClip("synthetic-shared.mp4", "shared");
      privateId = await uploadClip("synthetic-private.mp4", "private");
      const [first] = await transcribeAndDiarize(sharedId, {
        secondFingerprintEnd: 4.2,
      });
      await nameVoice(first, "Alice");
      await correctText(sharedId, 0, "Bien joué, Alice !", "nom");
      await correctSpeaker(sharedId, 1, first);
      await createApprovedMember();
    });

    test("la grille, les filtres et les visibilités", async ({ browser }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(ADMIN),
      });
      const page = await context.newPage();
      await page.goto("/library");
      await expect(page.getByText("2 items", { exact: true })).toBeVisible();
      const card = (name: string) =>
        page.getByRole("link", { name: `Open the clip ${name}` });
      await expect(card("synthetic-shared.mp4")).toBeVisible();
      await expect(card("synthetic-private.mp4")).toBeVisible();
      // A state has a label (and a symbol), not only a color: the transcribed clip is not "Not analysed".
      await expect(
        page.getByRole("table").getByText("Not analysed", { exact: true }),
      ).toHaveCount(1);

      // A filter lives in the address and narrows the list.
      await page.getByLabel("Owner").selectOption({ label: "administrator" });
      await expect(card("synthetic-private.mp4")).toBeVisible();
      await page.getByRole("button", { name: "Clear the filters" }).click();
      await expect(card("synthetic-private.mp4")).toBeVisible();

      await page.getByLabel("State").selectOption({ label: "Not indexed" });
      await expect(
        page.getByRole("heading", {
          name: "No clip matches these filters",
        }),
      ).toBeVisible();
      await page.getByRole("button", { name: "Clear the filters" }).click();

      await page.getByLabel("From").fill("2099-01-01");
      await expect(
        page.getByRole("heading", { name: "No clip matches these filters" }),
      ).toBeVisible();
      await context.close();
    });

    test("un membre ne voit que les clips partagés", async ({ browser }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(MEMBER),
      });
      const page = await context.newPage();
      await page.goto("/library");
      await expect(
        page.getByRole("link", { name: "Open the clip synthetic-shared.mp4" }),
      ).toBeVisible();
      await expect(page.getByText("1 item", { exact: true })).toBeVisible();
      await expect(
        page.getByRole("link", {
          name: "Open the clip synthetic-private.mp4",
        }),
      ).toHaveCount(0);
      await page.goto(`/clips/${privateId}`);
      await expect(page.getByRole("alert")).toContainText(
        "does not exist or is not visible",
      );
      await context.close();
    });

    test("la page d'un clip lit la transcription et garde le moment", async ({
      browser,
    }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(ADMIN),
      });
      const page = await context.newPage();
      await page.goto("/library");
      await page
        .getByRole("link", { name: "Open the clip synthetic-shared.mp4" })
        .click();
      await expect(page).toHaveURL(new RegExp(`/clips/${sharedId}$`));
      await expect(
        page.getByRole("heading", { level: 1, name: "synthetic-shared.mp4" }),
      ).toBeVisible();
      await expect(page.locator("video")).toHaveAttribute(
        "src",
        `/api/clips/${sharedId}/media`,
      );

      // Each segment carries its speaker and how settled it is.
      const segments = page
        .getByRole("list", { name: "Transcript segments" })
        .getByRole("listitem");
      await expect(segments).toHaveCount(SEGMENTS.length);
      await expect(segments.nth(0)).toContainText("Bien joué, Alice !");
      await expect(segments.nth(0)).toContainText("Alice");
      await expect(segments.nth(0)).toContainText("Identified");
      await expect(segments.nth(0)).toContainText("Text corrected by");
      await expect(segments.nth(1)).toContainText("Corrected");
      // The second voice only looks like Alice: a suggestion, not an identification.
      await expect(segments.nth(2)).toContainText("Suggestion");

      // The progress banner follows the real state of the clip.
      const progress = page.getByRole("list", { name: "Progress of the clip" });
      const step = (name: string) =>
        progress.getByRole("listitem").filter({ hasText: name });
      await expect(step("Imported")).toContainText("Done");
      await expect(step("Transcribed")).toContainText("Done");
      await expect(step("Indexed")).toContainText("Not set up");
      await expect(step("Voices processed")).toContainText("Done");
      await expect(step("Voices identified")).toContainText("1 of 2");

      // Playing from a segment keeps the moment in the address.
      await segments
        .nth(1)
        .getByRole("button", { name: "Play from 0:02" })
        .click();
      await expect(page).toHaveURL(/[?&]t=2(&|$)/);

      // An address with a moment opens the clip there.
      await page.goto(`/clips/${sharedId}?t=3`);
      await expect(segments.nth(1)).toHaveAttribute("aria-current", "true");
      await page.goto(`/clips/${sharedId}?t=2`);
      await expect(segments.nth(0)).toHaveAttribute("aria-current", "true");

      // Voices and history are read here too.
      await page.getByRole("tab", { name: "Voices" }).click();
      const voices = page.getByRole("list", { name: "Voices of the clip" });
      await expect(voices).toContainText("Alice");
      await expect(voices).toContainText("SPEAKER_01");
      await page.getByRole("tab", { name: "History" }).click();
      const corrected = page.getByRole("list", { name: "Corrected segments" });
      await expect(corrected.getByRole("listitem")).toHaveCount(2);
      await corrected
        .getByRole("button", { name: /Show the history/ })
        .first()
        .click();
      await expect(corrected).toContainText("Text change");
      await expect(corrected).toContainText("Reason: nom");
      await expect(corrected).toContainText("Bien joué !");

      // Back to the library.
      await page
        .getByRole("link", { name: "Library", exact: true })
        .first()
        .click();
      await expect(page).toHaveURL(/\/library/);
      await context.close();
    });

    test("un clip sans transcription le dit dans le bandeau", async ({
      browser,
    }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(ADMIN),
      });
      const page = await context.newPage();
      await page.goto(`/clips/${privateId}`);
      const progress = page.getByRole("list", { name: "Progress of the clip" });
      await expect(
        progress.getByRole("listitem").filter({ hasText: "Transcribed" }),
      ).toContainText("Not started");
      await expect(
        page.getByText("This clip has no transcript yet."),
      ).toBeVisible();
      await context.close();
    });

    test("la fiche : en-tête, lecteur et onglets côte à côte, métadonnées repliées", async ({
      browser,
    }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(ADMIN),
      });
      const page = await context.newPage();
      await page.goto(`/clips/${sharedId}`);
      await expect(
        page.getByRole("heading", { level: 1, name: "synthetic-shared.mp4" }),
      ).toBeVisible();
      // The file name is always in the header, as the title is still the file name.
      await expect(
        page.getByText("File: synthetic-shared.mp4", { exact: true }),
      ).toBeVisible();

      // The player and the tabs share the row: the tabs start after the player ends.
      const video = await page.locator("video").boundingBox();
      const tabs = await page.getByRole("tablist").boundingBox();
      expect(video).not.toBeNull();
      expect(tabs).not.toBeNull();
      expect(tabs!.x).toBeGreaterThanOrEqual(video!.x + video!.width);

      // Full metadata folds under the player and unfolds on demand.
      const toggle = page.getByRole("button", { name: "Full metadata" });
      await expect(toggle).toHaveAttribute("aria-expanded", "false");
      await expect(page.getByText("Codecs", { exact: true })).toBeHidden();
      await toggle.click();
      await expect(toggle).toHaveAttribute("aria-expanded", "true");
      await expect(page.getByText("Codecs", { exact: true })).toBeVisible();
      await expect(page.getByText("Format", { exact: true })).toBeVisible();
      await expect(page.getByText("Size", { exact: true })).toBeVisible();
      await expectNoViolations(page);

      // The transcription action lives in the Transcript tab, not stacked above the tabs.
      await expect(
        page.getByRole("tabpanel").getByRole("button", {
          name: "Transcribe again",
        }),
      ).toHaveCount(1);
      await context.close();
    });

    test("une fiche jamais analysée met le lancement en évidence", async ({
      browser,
    }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(ADMIN),
      });
      const page = await context.newPage();
      await page.goto(`/clips/${privateId}`);
      await expect(
        page.getByText("This clip has never been analysed.", { exact: true }),
      ).toBeVisible();
      // The launch is on the page itself, not behind the analysis pill.
      const launch = page.getByRole("button", { name: "Launch analysis…" });
      await expect(launch).toBeVisible();
      await launch.click();
      await expect(
        page.getByRole("dialog", { name: "Launch analysis" }),
      ).toBeVisible();
      await page.keyboard.press("Escape");
      await expect(
        page.getByRole("button", { name: "Transcribe again" }),
      ).toHaveCount(0);
      await expectNoViolations(page);
      await context.close();
    });

    for (const theme of ["light", "dark"] as const) {
      test(`accessibilité de la bibliothèque et du clip, thème ${theme}`, async ({
        browser,
      }) => {
        const context = await browser.newContext({
          storageState: await sessionFor(ADMIN),
        });
        await context.addInitScript((value) => {
          localStorage.setItem("snitch-ui-theme", value);
        }, theme);
        const page = await context.newPage();
        await page.goto("/library");
        await expect(
          page.getByRole("link", {
            name: "Open the clip synthetic-shared.mp4",
          }),
        ).toBeVisible();
        await expectNoViolations(page);
        await page.goto(`/clips/${sharedId}`);
        await expect(
          page.getByRole("list", { name: "Transcript segments" }),
        ).toBeVisible();
        await expectNoViolations(page);
        for (const tab of ["Voices", "History"]) {
          await page.getByRole("tab", { name: tab }).click();
          await expectNoViolations(page);
        }
        await context.close();
      });
    }

    test("téléphone : la grille et la page du clip tiennent à 390 pixels", async ({
      browser,
    }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(ADMIN),
        viewport: { width: 390, height: 780 },
      });
      const page = await context.newPage();
      await page.goto("/library");
      await expect(
        page.getByRole("link", { name: "Open the clip synthetic-shared.mp4" }),
      ).toBeVisible();
      expect(await noOverflow(page)).toBe(true);
      await expectNoViolations(page);
      await page.goto(`/clips/${sharedId}`);
      await expect(
        page.getByRole("list", { name: "Transcript segments" }),
      ).toBeVisible();
      expect(await noOverflow(page)).toBe(true);
      await expectNoViolations(page);
      await context.close();
    });

    // The explorer has no row action for the permanent deletion (ADR 0032): it stays an API action of
    // the Super-administrator, and the library must forget the clip afterwards.
    test("le super-administrateur supprime définitivement un clip, un membre ne le peut pas", async ({
      browser,
    }) => {
      const id = await uploadClip("synthetic-delete.mp4", "shared");
      const context = await browser.newContext({
        storageState: await sessionFor(ADMIN),
      });
      const page = await context.newPage();
      await page.goto("/library");
      await expect(page.getByText("3 items", { exact: true })).toBeVisible();
      await expect(
        page.getByRole("link", { name: "Open the clip synthetic-delete.mp4" }),
      ).toBeVisible();

      const memberApi = await request.newContext({
        baseURL: process.env.SNITCH_TEST_URL!,
        extraHTTPHeaders: { Origin: process.env.SNITCH_TEST_URL! },
        storageState: await sessionFor(MEMBER),
      });
      const memberMe = await (await memberApi.get("/api/auth/me")).json();
      const refused = await memberApi.delete(`/api/admin/clips/${id}`, {
        headers: { "X-CSRF-Token": memberMe.csrf_token },
      });
      expect(refused.status()).toBe(403);
      await memberApi.dispose();

      const me = await (await page.request.get("/api/auth/me")).json();
      const deleted = await page.request.delete(`/api/admin/clips/${id}`, {
        headers: {
          "X-CSRF-Token": me.csrf_token,
          Origin: process.env.SNITCH_TEST_URL!,
        },
      });
      expect(deleted.status()).toBe(200);

      await page.reload();
      await expect(page.getByText("2 items", { exact: true })).toBeVisible();
      await expect(
        page.getByRole("link", { name: "Open the clip synthetic-delete.mp4" }),
      ).toHaveCount(0);
      await context.close();
    });

    // The explorer lists a folder at once (no server pagination, ADR 0032): every clip is on the page.
    test("l'explorateur liste plus de douze clips d'un coup, sans pagination", async ({
      browser,
    }) => {
      for (let i = 1; i <= 11; i++) {
        await uploadClip(`synthetic-bulk-${i}.mp4`, "shared");
      }
      const context = await browser.newContext({
        storageState: await sessionFor(ADMIN),
      });
      const page = await context.newPage();
      await page.goto("/library");
      await expect(page.getByText("13 items", { exact: true })).toBeVisible();
      await expect(
        page.getByRole("link", { name: /^Open the clip / }),
      ).toHaveCount(13);
      await expect(page.getByRole("button", { name: "Next page" })).toHaveCount(
        0,
      );
      await context.close();
    });

    test("un administrateur modifie titre, jeu et date ; les corrections se restaurent", async ({
      browser,
    }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(ADMIN),
      });
      const page = await context.newPage();
      await page.goto(`/clips/${sharedId}`);
      await expect(
        page.getByRole("heading", { level: 1, name: "synthetic-shared.mp4" }),
      ).toBeVisible();

      // Members see no editing action at all.
      const member = await browser.newContext({
        storageState: await sessionFor(MEMBER),
      });
      const memberPage = await member.newPage();
      await memberPage.goto(`/clips/${sharedId}`);
      await expect(memberPage.getByRole("heading", { level: 1 })).toBeVisible();
      await expect(
        memberPage.getByRole("button", { name: "Edit metadata" }),
      ).toHaveCount(0);
      await member.close();

      await page.getByRole("button", { name: "Edit metadata" }).click();
      const dialog = page.getByRole("dialog", { name: "Edit metadata" });
      await expect(dialog.getByLabel("Displayed title")).toHaveValue(
        "synthetic-shared.mp4",
      );
      await expectNoViolations(page);
      // An empty title is refused with an explanation; nothing is sent.
      await dialog.getByLabel("Displayed title").fill("   ");
      await dialog.getByRole("button", { name: "Save" }).click();
      await expect(dialog.getByRole("alert")).toContainText("cannot be empty");

      await dialog.getByLabel("Displayed title").fill("Clutch 1v4 — Mirage 🎮");
      await dialog.getByLabel("Game").fill("Counter-Strike 2");
      await dialog.getByLabel("Recording date").fill("2026-10-31T21:45");
      await dialog.getByRole("button", { name: "Save" }).click();
      await expect(dialog).toBeHidden();

      // The page shows the title at once, keeps the file name, and tells what was corrected by hand.
      await expect(
        page.getByRole("heading", { level: 1, name: "Clutch 1v4 — Mirage 🎮" }),
      ).toBeVisible();
      await expect(
        page.getByText("File name: synthetic-shared.mp4 (unchanged)"),
      ).toBeVisible();
      await expect(page.getByText("Manual correction")).toHaveCount(2);
      await expect(page).toHaveTitle(/Clutch 1v4/);

      // The library uses the title, the file name stays reachable as technical information.
      await page.goto("/library");
      await expect(
        page.getByRole("link", {
          name: "Open the clip Clutch 1v4 — Mirage 🎮",
        }),
      ).toBeVisible();
      await expect(
        page.getByRole("link", { name: "Open the clip synthetic-shared.mp4" }),
      ).toHaveCount(0);

      // Restoring the detected game puts the automatic value back; the file name restores the title.
      await page.goto(`/clips/${sharedId}`);
      await page.getByRole("button", { name: "Edit metadata" }).click();
      await expect(dialog.getByText("Detected value:").first()).toBeVisible();
      await dialog.getByRole("button", { name: "Restore" }).first().click();
      await dialog.getByRole("button", { name: "Use the file name" }).click();
      await expectNoViolations(page);
      await dialog.getByRole("button", { name: "Save" }).click();
      await expect(dialog).toBeHidden();
      await expect(
        page.getByRole("heading", { level: 1, name: "synthetic-shared.mp4" }),
      ).toBeVisible();
      await expect(page.getByText("Manual correction")).toHaveCount(1);

      // On a phone the panel still fits.
      await page.setViewportSize({ width: 390, height: 800 });
      await page.getByRole("button", { name: "Edit metadata" }).click();
      await expect(dialog).toBeVisible();
      expect(await noOverflow(page)).toBe(true);
      await page.keyboard.press("Escape");
      await expect(dialog).toBeHidden();
      await context.close();
    });
  });
});
