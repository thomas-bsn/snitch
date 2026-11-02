import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { adminApi, transcribeAndDiarize, uploadClip } from "./library-fixture";
import { createPersons } from "./people-fixture";
import { prepareDiarizedClip } from "./voice-fixture";

// The People screen of the new interface: the list of Persons with its corrections, and the
// Verification of unknown voices. The clip, its two close synthetic voices and the session
// come from the shared fixture (the instance is installed through the guided setup);
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
let clipId: number;

test.describe.serial("personnes et vérification", () => {
  // The installation code works once: the three journeys share one prepared instance and
  // follow one another (the first settles a voice, the second corrects it).
  test.beforeAll(async ({ browser }) => {
    page = await (await browser.newContext()).newPage();
    clipId = await prepareDiarizedClip(page, { secondFingerprintEnd: 4.2 });
  });
  test.afterAll(async () => {
    await page.context().close();
  });

  test("vérification : un cas à l'écran, passer en local, choisir au clavier et trancher", async () => {
    await page.goto("/people/verification");
    await expect(page.getByRole("heading", { level: 1 })).toHaveText("People");
    await expect(
      page.getByRole("link", { name: /^Verification\s*2$/ }),
    ).toBeVisible();

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
    // One case at a time, the most frequent first.
    const current = page.getByRole("region", { name: "Voice to decide" });
    await expect(current.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_00",
    );
    await expect(current).toContainText("2 voice(s) left to decide");
    await expect(current).toContainText("2 voice(s) in the group");
    await expect(
      page.getByRole("list", { name: "Voices to decide" }),
    ).toHaveCount(0);
    await expectNoViolations(page);
    // A badge placed inside a button is not a button itself.
    await expect(page.locator("button button, a button")).toHaveCount(0);

    // A counter filters the queue.
    await counters.getByRole("button", { name: /^Ambiguous/ }).click();
    await expect(page.getByText("Nothing left to decide here.")).toBeVisible();
    await counters.getByRole("button", { name: /^To decide/ }).click();
    await expect(current).toContainText("2 voice(s) left to decide");

    // The excerpt is listened to in place and served to the session only.
    await expect(
      current.getByRole("button", {
        name: "Excerpt of SPEAKER_00 in synthetic-voices.mp4",
      }),
    ).toBeVisible();
    const source = await current.locator("audio").getAttribute("src");
    expect(source).toMatch(/^\/api\/local-speakers\/\d+\/excerpt$/);
    expect((await page.request.get(source!)).headers()["content-type"]).toBe(
      "audio/wav",
    );

    // "Confirm" stays off until something is chosen, and Enter then does nothing.
    const confirm = current.getByRole("button", { name: "Confirm" });
    await expect(confirm).toBeDisabled();
    await page.keyboard.press("Enter");
    await expect(current.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_00",
    );
    await current.getByRole("radio", { name: "Create a person" }).check();
    await expect(confirm).toBeDisabled();

    // "Skip" is local: the next case comes, the counters do not move, and a reload
    // brings the original order back.
    await page.keyboard.press("s");
    await expect(current.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_01",
    );
    await expect(
      counters.getByRole("button", { name: /^Unknown\s*2$/ }),
    ).toBeVisible();
    await page.reload();
    await expect(current.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_00",
    );
    await expect(
      counters.getByRole("button", { name: /^Unknown\s*2$/ }),
    ).toBeVisible();

    // The shortcuts are off in a text field: the name holds an "s" and "Enter".
    await current.getByRole("radio", { name: "Create a person" }).check();
    const name = current.getByLabel("Name of the new person");
    await name.fill("Claire Synthétique");
    await name.press("Enter");
    await expect(current.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_00",
    );
    await expect(name).toHaveValue("Claire Synthétique");
    await expectNoViolations(page);

    // Settling creates the person, removes the case and lowers the counters.
    await confirm.click();
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
    await expect(
      page.getByRole("link", { name: /^Verification\s*0$/ }),
    ).toBeVisible();
  });

  test("badges de niveau avec légende et extraits écoutables sur la fiche", async () => {
    // The synthetic excerpt cannot be decoded: the media element is stubbed, and what is
    // observed is which excerpt was asked for and which one was stopped.
    await page.addInitScript(() => {
      const log: string[] = [];
      (window as unknown as { played: string[] }).played = log;
      HTMLMediaElement.prototype.play = function () {
        log.push(this.getAttribute("src") ?? "");
        this.dispatchEvent(new Event("play"));
        return Promise.resolve();
      };
      HTMLMediaElement.prototype.pause = function () {
        this.dispatchEvent(new Event("pause"));
      };
    });
    // A second voice of the clip is confirmed for the same Person.
    await page.goto("/people");
    await page
      .getByRole("list", { name: "People", exact: true })
      .getByRole("link", { name: /Claire Synthétique/ })
      .click();
    await expect(page).toHaveURL(/\/people\/\d+$/);
    const personId = Number(page.url().split("/").pop());
    const { api, post } = await adminApi();
    const voice = (await (
      await api.get(`/api/clips/${clipId}/voice`)
    ).json()) as { speakers: { id: number; label: string }[] };
    const second = voice.speakers.find((s) => s.label === "SPEAKER_01")!;
    await post(`/api/local-speakers/${second.id}/validation`, {
      person_id: personId,
    });
    await api.dispose();
    await page.reload();

    const section = page.getByRole("group", {
      name: "Confirmed voices of Claire Synthétique",
    });
    await expect(
      section.getByRole("heading", { name: "Identifications" }),
    ).toBeVisible();
    const items = section.getByRole("listitem");
    await expect(items).toHaveCount(2);

    // The badge is a button: hovering alone shows nothing, Enter opens the legend.
    const badge = items.first().getByRole("button", {
      name: "Validated by a human",
    });
    const legend = page.getByRole("dialog", { name: "Validated by a human" });
    await badge.hover();
    await expect(legend).toHaveCount(0);
    await badge.focus();
    await page.keyboard.press("Enter");
    await expect(legend).toBeVisible();
    await expect(legend).toContainText("Counts as a Reference: yes");
    for (const other of [
      "Automatic attribution",
      "Suggestion",
      "Ambiguous",
      "Unknown",
    ]) {
      await expect(legend).toContainText(other);
    }
    await expectNoViolations(page);
    // Escape closes it and the focus is back on the badge.
    await page.keyboard.press("Escape");
    await expect(legend).toHaveCount(0);
    await expect(badge).toBeFocused();
    // A click opens it, a second click and a click outside close it.
    await badge.click();
    await expect(legend).toBeVisible();
    await badge.click();
    await expect(legend).toHaveCount(0);
    await badge.click();
    await expect(legend).toBeVisible();
    await page.getByRole("heading", { level: 1 }).click();
    await expect(legend).toHaveCount(0);

    // The excerpt played is the right one, and a second one stops the first.
    const players = items.getByRole("button", {
      name: /^Excerpt of SPEAKER_0/,
    });
    const first = players.nth(0);
    const other = players.nth(1);
    await first.click();
    await expect(first).toHaveAttribute("aria-pressed", "true");
    const played = () =>
      page.evaluate(() => (window as unknown as { played: string[] }).played);
    expect(await played()).toEqual([
      expect.stringMatching(/^\/api\/local-speakers\/\d+\/excerpt$/),
    ]);
    await other.click();
    await expect(other).toHaveAttribute("aria-pressed", "true");
    await expect(first).toHaveAttribute("aria-pressed", "false");
    expect((await played()).length).toBe(2);
    // The way to the clip.
    await expect(
      items.first().getByRole("link", { name: "Open synthetic-voices.mp4" }),
    ).toHaveAttribute("href", `/clips/${clipId}`);

    // The second voice goes back to where it was, for the journeys that follow.
    const cleanup = await adminApi();
    await cleanup.post(`/api/persons/${personId}/separation`, {
      local_speaker_ids: [second.id],
    });
    await cleanup.api.dispose();
  });

  test("personnes : liste, fiche, création, fusion, séparation et suppression", async () => {
    await page.goto("/people");
    const list = page.getByRole("list", { name: "People", exact: true });
    await expect(list).toContainText("Claire Synthétique");
    await expect(page.getByText("No person yet")).toHaveCount(0);
    await expect(page.getByRole("table")).toHaveCount(0);

    // The record of a Person shows its real counts.
    await list.getByRole("link", { name: /Claire Synthétique/ }).click();
    const claire = page.getByRole("region", { name: "Claire Synthétique" });
    await expect(claire).toContainText("1 voice(s)");
    await expect(claire).toContainText("1 validated sample(s)");
    const claireUrl = page.url();

    // Creation.
    await page.getByRole("button", { name: "New person" }).click();
    const creation = page.getByRole("dialog", { name: "New person" });
    await creation.getByLabel("Name").fill("Alicia Synthétique");
    await expectNoViolations(page);
    await creation.getByRole("button", { name: "Create" }).click();
    await expect(page.getByRole("status")).toContainText(
      "Alicia Synthétique was created.",
    );
    await expect(list).toContainText("Alicia Synthétique");
    await expectNoViolations(page);

    // The list is searchable; a Person without a validated sample has no reference.
    await page.getByRole("searchbox", { name: "Search people" }).fill("alicia");
    await expect(list.getByRole("listitem")).toHaveCount(1);
    await expect(list).toContainText("No reference");
    await page.getByRole("searchbox", { name: "Search people" }).fill("");
    await list.getByRole("link", { name: /Alicia Synthétique/ }).click();
    const alicia = page.getByRole("region", { name: "Alicia Synthétique" });
    await expect(alicia).toContainText("No reference");

    // Merge: nothing happens until the confirmation, then the merged person disappears.
    await alicia
      .getByRole("button", { name: "Merge Alicia Synthétique" })
      .click();
    const merge = page.getByRole("dialog", {
      name: "Merge Alicia Synthétique",
    });
    await expect(
      merge.getByRole("button", { name: "Merge", exact: true }),
    ).toBeDisabled();
    await merge.getByRole("searchbox", { name: "Merge into" }).fill("claire");
    await merge.getByRole("radio", { name: "Claire Synthétique" }).check();
    await expectNoViolations(page);
    await merge.getByRole("button", { name: "Merge", exact: true }).click();
    await expect(page.getByRole("status")).toContainText(
      "Alicia Synthétique was merged into Claire Synthétique",
    );
    await expect(page).toHaveURL(claireUrl);
    await expect(list).not.toContainText("Alicia Synthétique");

    // Separation: the voices are on the record, a cancelled then a confirmed separation.
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
    await expect(claire).toContainText("0 voice(s)");

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
      page
        .getByRole("region", { name: "Voice to decide" })
        .getByRole("heading", { level: 2 }),
    ).toHaveText("SPEAKER_00");

    // Deletion, from the address of the record, after a confirmation.
    await page.goto(claireUrl);
    await page
      .getByRole("button", { name: "Delete Claire Synthétique" })
      .click();
    const deletion = page.getByRole("alertdialog", {
      name: "Delete Claire Synthétique?",
    });
    await expectNoViolations(page);
    await deletion.getByRole("button", { name: "Cancel" }).click();
    await expect(claire).toBeVisible();
    await page
      .getByRole("button", { name: "Delete Claire Synthétique" })
      .click();
    await deletion.getByRole("button", { name: "Delete", exact: true }).click();
    await expect(page.getByRole("status")).toContainText(
      "Claire Synthétique was deleted",
    );
    await expect(page).toHaveURL(/\/people$/);
    await expect(page.getByText("No person yet")).toBeVisible();
    await page.goto(claireUrl);
    await expect(page.getByText("Person not found")).toBeVisible();
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
    const confirm = page
      .getByRole("region", { name: "Voice to decide" })
      .getByRole("button", { name: "Confirm" });
    await expect(confirm).toBeVisible();
    expect((await confirm.boundingBox())!.height).toBeGreaterThanOrEqual(44);
    await expectNoViolations(page);
  });

  test("la légende des badges reste lisible en thème sombre à 390 pixels", async () => {
    await page.goto("/people/verification");
    const caseBadge = page
      .getByRole("region", { name: "Voice to decide" })
      .getByRole("button", { name: "Unknown" });
    await caseBadge.click();
    const legend = page.getByRole("dialog", { name: "Unknown" });
    await expect(legend).toBeVisible();
    await expect(legend).toContainText("Counts as a Reference: no");
    const box = (await legend.boundingBox())!;
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(390);
    await expectNoViolations(page);
  });

  test("vérification : recherche serveur, enchaînement, Entrée au clavier et cas déjà tranché", async () => {
    // Twelve more Persons than a page of results holds, and a second clip with new voices.
    const names = Array.from(
      { length: 12 },
      (_, i) => `Zoé Synthétique ${String(i + 1).padStart(2, "0")}`,
    );
    const ids = await createPersons(names);
    const second = await uploadClip("second-voices.mp4", "shared");
    await transcribeAndDiarize(second);
    await page.evaluate(() => localStorage.setItem("snitch-ui-theme", "light"));
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto("/people/verification");

    const current = page.getByRole("region", { name: "Voice to decide" });
    const heading = current.getByRole("heading", { level: 2 });
    const tab = page.getByRole("link", { name: /^Verification\s*\d+$/ });
    await expect(heading).toBeVisible();
    const count = async () =>
      Number((await tab.innerText()).replace(/\D/g, ""));
    const before = await count();
    expect(before).toBeGreaterThanOrEqual(2);
    const speakerOf = async () =>
      Number(
        (await current.locator("audio").getAttribute("src"))!.match(
          /local-speakers\/(\d+)\//,
        )![1],
      );

    // With nothing chosen, a number key changes nothing.
    const confirm = current.getByRole("button", { name: "Confirm" });
    await page.keyboard.press("9");
    await expect(confirm).toBeDisabled();

    // The search is made by the server: the eleventh person is found past the first page.
    const search = current.getByRole("searchbox", { name: "Search people" });
    await search.fill("Synthétique 11");
    const eleven = current.getByRole("radio", { name: "Zoé Synthétique 11" });
    await eleven.check();
    await expect(confirm).toBeEnabled();
    const first = await speakerOf();
    // Enter, with the focus on the choice, confirms; the next case follows by itself.
    await page.keyboard.press("Enter");
    await expect(page.getByRole("status")).toContainText(
      "is Zoé Synthétique 11.",
    );
    await expect.poll(speakerOf).not.toBe(first);
    await expect.poll(count).toBeLessThan(before);

    // A candidate, if the server proposes one, is chosen with its number and shows a
    // proximity, never a confirmation.
    const queue = (await (
      await page.request.get("/api/voice-verification?level=all")
    ).json()) as {
      items: { candidates: { person: { display_name: string } }[] }[];
    };
    const candidates = queue.items[0]?.candidates ?? [];
    if (candidates.length > 0) {
      await expect(current).toContainText("% proximity");
      await page.keyboard.press("1");
      await expect(
        current.getByRole("radio", { name: candidates[0].person.display_name }),
      ).toBeChecked();
      await expect(confirm).toBeEnabled();
      await expect(current).toContainText(
        "Proximity is only a suggestion: nothing is attributed until you confirm.",
      );
    }

    // Somebody else settles the displayed case first: the choice is refused, nothing is
    // written for the chosen Person and the queue goes on.
    const target = await speakerOf();
    const { api, post } = await adminApi();
    await post(`/api/local-speakers/${target}/validation`, {
      person_id: ids[0],
    });
    await search.fill("Synthétique 12");
    await current.getByRole("radio", { name: "Zoé Synthétique 12" }).check();
    await confirm.click();
    await expect(page.getByRole("status")).toContainText(
      "Already confirmed as Zoé Synthétique 01.",
    );
    await expect(page.getByText("Nothing left to decide here.")).toBeVisible();
    await expect.poll(count).toBe(0);
    const twelve = (await (
      await api.get(`/api/persons/${ids[11]}`)
    ).json()) as { local_speaker_count: number };
    expect(twelve.local_speaker_count).toBe(0);
    await api.dispose();
    await expectNoViolations(page);
  });
});
