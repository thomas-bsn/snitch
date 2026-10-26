import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// The People screen of the new interface: the list of Persons with its corrections, and the
// Verification of unknown voices. The clip, its two close synthetic voices and the session
// come from the shared fixture (the instance is installed through the previous interface);
// everything under test goes through the new screens.

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

let page: Page;

test.describe.serial("personnes et vérification", () => {
  // The installation code works once: the three journeys share one prepared instance and
  // follow one another (the first settles a voice, the second corrects it).
  test.beforeAll(async ({ browser }) => {
    page = await (await browser.newContext()).newPage();
    await prepareDiarizedClip(page, { secondFingerprintEnd: 4.2 });
  });
  test.afterAll(async () => {
    await page.context().close();
  });

  test("vérification : compteurs, extrait, panneau Décider, passer et trancher", async () => {
    await page.goto("/people/verification");
    await expect(page.getByRole("heading", { level: 1 })).toHaveText("People");

    // The five states carry a label and the counts come from the server.
    const counters = page.getByRole("list", { name: "Voices by state" });
    await expect(
      counters.getByRole("button", { name: /^Unknown\s*2$/ }),
    ).toBeVisible();
    await expect(
      counters.getByRole("button", { name: /^Ambiguous\s*0$/ }),
    ).toBeVisible();
    for (const label of [
      "Validated by a human",
      "Automatic attribution",
      "Suggestion",
    ]) {
      await expect(
        counters.locator("li").filter({ hasText: label }),
      ).toHaveCount(1);
    }
    const cases = page.getByRole("list", { name: "Voices to decide" });
    await expect(cases.locator("xpath=./li")).toHaveCount(2);
    await expect(cases).toContainText("2 voice(s) in the group");
    await expectNoViolations(page);

    // A counter filters the list.
    await counters.getByRole("button", { name: /^Ambiguous/ }).click();
    await expect(page.getByText("Nothing left to decide here.")).toBeVisible();
    await counters.getByRole("button", { name: /^To decide/ }).click();
    await expect(cases.locator("xpath=./li")).toHaveCount(2);

    // The thumbnail opens the excerpt, which is served to the session only.
    await page
      .getByRole("button", {
        name: "Play the excerpt of SPEAKER_00 in synthetic-voices.mp4",
      })
      .click();
    const preview = page.getByRole("dialog", { name: "Excerpt of SPEAKER_00" });
    const source = await preview.locator("audio").getAttribute("src");
    expect(source).toMatch(/^\/api\/local-speakers\/\d+\/excerpt$/);
    expect((await page.request.get(source!)).headers()["content-type"]).toBe(
      "audio/wav",
    );
    await page.keyboard.press("Escape");
    await expect(preview).toHaveCount(0);

    // "Confirm" stays off until something is chosen, "Skip" changes nothing.
    const decide = () =>
      page.getByRole("button", { name: "Decide who SPEAKER_00 is" });
    await decide().click();
    const panel = page.getByRole("dialog", { name: "Who is SPEAKER_00?" });
    await expect(panel.getByRole("button", { name: "Confirm" })).toBeDisabled();
    await panel.getByRole("radio", { name: "Create a person" }).check();
    await expect(panel.getByRole("button", { name: "Confirm" })).toBeDisabled();
    await panel.getByRole("button", { name: "Skip for now" }).click();
    await expect(panel).toHaveCount(0);
    await expect(cases.locator("xpath=./li")).toHaveCount(2);
    await expect(
      counters.getByRole("button", { name: /^Unknown\s*2$/ }),
    ).toBeVisible();

    // Settling creates the person, removes the case and updates the counters.
    await decide().click();
    await panel.getByRole("radio", { name: "Create a person" }).check();
    await panel.getByLabel("Name of the new person").fill("Claire Synthétique");
    await expectNoViolations(page);
    await panel.getByRole("button", { name: "Confirm" }).click();
    await expect(page.getByRole("status")).toContainText(
      "Saved: SPEAKER_00 is Claire Synthétique.",
    );
    await expect(page.getByText("Nothing left to decide here.")).toBeVisible();
    await expect(
      counters.locator("li").filter({ hasText: "Validated by a human" }),
    ).toContainText("1");
    await expect(
      counters.locator("li").filter({ hasText: "Suggestion" }),
    ).toContainText("1");
    await expect(
      counters.getByRole("button", { name: /^Unknown\s*0$/ }),
    ).toBeVisible();
  });

  test("personnes : création, fusion, séparation et suppression", async () => {
    await page.goto("/people");
    const table = page.getByRole("table", { name: "People" });
    await expect(table).toContainText("Claire Synthétique");
    await expect(page.getByText("No person yet")).toHaveCount(0);

    // Creation.
    await page.getByRole("button", { name: "New person" }).click();
    const creation = page.getByRole("dialog", { name: "New person" });
    await creation.getByLabel("Name").fill("Alicia Synthétique");
    await expectNoViolations(page);
    await creation.getByRole("button", { name: "Create" }).click();
    await expect(page.getByRole("status")).toContainText(
      "Alicia Synthétique was created.",
    );
    await expect(table).toContainText("Alicia Synthétique");
    await expectNoViolations(page);

    // The list is searchable.
    await page.getByRole("searchbox", { name: "Search people" }).fill("alicia");
    await expect(table.locator("tbody tr")).toHaveCount(1);
    await page.getByRole("searchbox", { name: "Search people" }).fill("");

    // Merge: nothing happens until the confirmation, then the merged person disappears.
    await page
      .getByRole("button", { name: "Merge Alicia Synthétique" })
      .click();
    const merge = page.getByRole("dialog", {
      name: "Merge Alicia Synthétique",
    });
    await expect(
      merge.getByRole("button", { name: "Merge", exact: true }),
    ).toBeDisabled();
    await merge
      .getByLabel("Merge into")
      .selectOption({ label: "Claire Synthétique" });
    await expectNoViolations(page);
    await merge.getByRole("button", { name: "Merge", exact: true }).click();
    await expect(page.getByRole("status")).toContainText(
      "Alicia Synthétique was merged into Claire Synthétique",
    );
    await expect(table).not.toContainText("Alicia Synthétique");

    // Separation: the voice list, a cancelled then a confirmed separation.
    await page
      .getByRole("button", { name: "Voices of Claire Synthétique" })
      .click();
    const voices = page.getByRole("group", {
      name: "Confirmed voices of Claire Synthétique",
    });
    await expect(voices).toContainText("SPEAKER_00");
    await expectNoViolations(page);
    await voices
      .getByRole("button", {
        name: "Separate SPEAKER_00 from Claire Synthétique",
      })
      .click();
    const confirmation = page.getByRole("alertdialog", {
      name: "Separate this voice?",
    });
    await confirmation.getByRole("button", { name: "Cancel" }).click();
    await expect(voices).toContainText("SPEAKER_00");
    await voices
      .getByRole("button", {
        name: "Separate SPEAKER_00 from Claire Synthétique",
      })
      .click();
    await confirmation.getByRole("button", { name: "Separate" }).click();
    await expect(page.getByRole("status")).toContainText(
      "SPEAKER_00 was separated from Claire Synthétique: it is unknown again.",
    );
    await expect(voices).toContainText("No confirmed voice you can see.");

    // The corrections are kept in the log.
    await page.getByRole("button", { name: "Corrections log" }).click();
    const log = page.getByRole("list", { name: "Person corrections" });
    await expect(log).toContainText(
      "Merge: Alicia Synthétique → Claire Synthétique",
    );
    await expect(log).toContainText("Separation: Claire Synthétique");

    // The separated voice is back in the verification.
    await page.goto("/people/verification");
    await expect(
      page.getByRole("button", { name: "Decide who SPEAKER_00 is" }),
    ).toBeVisible();

    // Deletion, after a confirmation.
    await page.goto("/people");
    await page
      .getByRole("button", { name: "Delete Claire Synthétique" })
      .click();
    const deletion = page.getByRole("alertdialog", {
      name: "Delete Claire Synthétique?",
    });
    await expectNoViolations(page);
    await deletion.getByRole("button", { name: "Cancel" }).click();
    await expect(table).toContainText("Claire Synthétique");
    await page
      .getByRole("button", { name: "Delete Claire Synthétique" })
      .click();
    await deletion.getByRole("button", { name: "Delete", exact: true }).click();
    await expect(page.getByRole("status")).toContainText(
      "Claire Synthétique was deleted",
    );
    await expect(page.getByText("No person yet")).toBeVisible();
  });

  test("personnes et vérification en thème sombre et à 390 pixels", async () => {
    await page.evaluate(() => localStorage.setItem("snitch-ui-theme", "dark"));
    await page.setViewportSize({ width: 390, height: 780 });
    for (const path of ["/people", "/people/verification"]) {
      await page.goto(path);
      await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
      await expectNoViolations(page);
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= window.innerWidth,
        ),
      ).toBe(true);
    }
    await page
      .getByRole("button", { name: "Decide who SPEAKER_00 is" })
      .click();
    await expectNoViolations(page);
  });
});
