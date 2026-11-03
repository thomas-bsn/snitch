import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import {
  adminApi,
  nameVoice,
  transcribeAndDiarize,
  uploadClip,
} from "./library-fixture";
import { createPersons, linkAccount } from "./people-fixture";
import { createApprovedMember, MEMBER, sessionFor } from "./shell-fixture";
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

// The member account is created on demand by whichever test needs it first: no test
// relies on another one having run before it.
let memberReady: Promise<void> | undefined;
function ensureMember() {
  memberReady ??= createApprovedMember();
  return memberReady;
}

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

  test("traitement : une voix à l'écran, extrait écoutable, adresse d'un cas et voix tranchée", async () => {
    await page.goto("/people/verification");
    await expect(page.getByRole("heading", { level: 1 })).toHaveText("People");
    await expect(
      page.getByRole("link", { name: /^To process\s*2$/ }),
    ).toBeVisible();
    await page.getByRole("link", { name: /^Process / }).click();
    await expect(page).toHaveURL(
      new RegExp(`/people/verification\\?clip=${clipId}$`),
    );

    // One voice at a time, in the order of the clip; the queue shows both.
    const screen = page.getByRole("region", { name: "Clip processing" });
    const current = screen.getByRole("region", {
      name: "Who is speaking here?",
    });
    await expect(current.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_00",
    );
    await expect(current).toContainText("Voice 1 of 2");
    await expect(
      screen
        .getByRole("list", { name: "Voices to process" })
        .getByRole("listitem"),
    ).toHaveCount(2);
    await expectNoViolations(page);
    // A badge placed inside a button is not a button itself.
    await expect(page.locator("button button, a button")).toHaveCount(0);

    // A direct link opens its voice first, while the clip alone starts with SPEAKER_00.
    const { api: voiceApi } = await adminApi();
    const voices = (await (
      await voiceApi.get(`/api/clips/${clipId}/voice`)
    ).json()) as { speakers: { id: number; label: string }[] };
    await voiceApi.dispose();
    const idOf = (label: string) =>
      voices.speakers.find((s) => s.label === label)!.id;
    await page.goto(`/people/verification?case=${idOf("SPEAKER_01")}`);
    await expect(current.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_01",
    );
    await expectNoViolations(page);
    await page.goto(`/people/verification?clip=${clipId}`);
    await expect(current.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_00",
    );

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

    // Identifying with a new Person settles the voice and lowers the counter of the tab.
    await current.getByRole("button", { name: "Identify" }).click();
    await current
      .getByLabel("Name of the new person")
      .fill("Claire Synthétique");
    await current.getByRole("button", { name: "Create and identify" }).click();
    await expect(current.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_01",
    );
    await expect(
      page.getByRole("link", { name: /^To process\s*1$/ }),
    ).toBeVisible();
    // The sidebar follows the decision without a change of address; the voice left is
    // only a suggestion, and it still counts: the list shows it.
    await expect(
      page
        .getByRole("navigation", { name: "Main navigation" })
        .first()
        .getByRole("link", { name: "People" }),
    ).toHaveAccessibleDescription("1 voice(s) to decide");

    // The address of the voice just settled opens its clip, with the voice marked settled.
    await page.goto(`/people/verification?case=${idOf("SPEAKER_00")}`);
    await expect(current.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_00",
    );
    await expect(current).toContainText("Validated by a person");
    await expect(
      screen.getByRole("list", { name: "Voices to process" }),
    ).toContainText("Claire Synthétique");
    // The voice is already settled: the notice says who attributed it, to whom, when.
    const handled = page.getByRole("region", {
      name: "Direct link to a voice",
    });
    await expect(handled.getByRole("heading", { level: 2 })).toHaveText(
      "Already handled",
    );
    await expect(handled).toContainText(
      /SPEAKER_00 was attributed to Claire Synthétique by .+ on .+\./,
    );
    await expect(
      handled.getByRole("link", { name: "Open Claire Synthétique" }),
    ).toHaveAttribute("href", /^\/people\/\d+$/);
    await expectNoViolations(page);
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
    // The right margin is free whatever the height of the record puts under the legend.
    await page.mouse.click(1275, 360);
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
    await expect(claire).toContainText("1 video(s)");
    await expect(claire.getByRole("note")).toContainText(
      "Reference to strengthen",
    );
    await expect(
      claire
        .getByRole("group", {
          name: "Videos where Claire Synthétique is heard",
        })
        .getByRole("link"),
    ).toHaveCount(1);
    const claireUrl = page.url();

    // Creation, from the directory.
    await page.getByRole("link", { name: "Back to people" }).click();
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
    await expect(list).toContainText("Reference: none");
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
    // The consequences are counted, with what stays intact; no name to type.
    await expect(merge).toContainText(
      "0 confirmed voice(s) of Alicia Synthétique are now confirmed as Claire Synthétique.",
    );
    await expect(merge).toContainText("0 validated sample(s) of Alicia");
    await expect(merge).toContainText("0 microphone track rule(s) naming");
    await expect(merge).toContainText("The clips, the voice fingerprints");
    await expect(merge.getByRole("textbox")).toHaveCount(0);
    await expectNoViolations(page);
    await merge.getByRole("button", { name: "Merge", exact: true }).click();
    await expect(page.getByRole("status")).toContainText(
      "Alicia Synthétique was merged into Claire Synthétique",
    );
    await expect(page).toHaveURL(claireUrl);
    await page.getByRole("link", { name: "Back to people" }).click();
    await expect(list).not.toContainText("Alicia Synthétique");
    await list.getByRole("link", { name: /Claire Synthétique/ }).click();

    // Separation: an action of the header, which counts what it does.
    const voices = page.getByRole("group", {
      name: "Confirmed voices of Claire Synthétique",
    });
    await expect(voices).toContainText("SPEAKER_00");
    await expect(voices.getByRole("button", { name: /^Separate/ })).toHaveCount(
      0,
    );
    await expectNoViolations(page);
    const separate = claire.getByRole("button", {
      name: "Separate voices of Claire Synthétique",
    });
    await expect(separate).toHaveAttribute("aria-disabled", "false");
    await separate.click();
    const separation = page.getByRole("dialog", {
      name: "Separate voices of Claire Synthétique",
    });
    const confirm = separation.getByRole("button", {
      name: "Separate",
      exact: true,
    });
    await expect(confirm).toBeDisabled();
    await separation.getByRole("checkbox", { name: /SPEAKER_00/ }).check();
    await expect(separation).toContainText("1 voice(s) become unknown again");
    await expect(separation).toContainText(
      "1 validated sample(s) leave the references",
    );
    await expect(separation).toContainText("Claire Synthétique, the clips");
    await expectNoViolations(page);
    await separation.getByRole("button", { name: "Cancel" }).click();
    await expect(voices).toContainText("SPEAKER_00");
    await separate.click();
    await separation.getByRole("checkbox", { name: /SPEAKER_00/ }).check();
    await confirm.click();
    await expect(page.getByRole("status")).toContainText(
      "1 voice was separated from Claire Synthétique: it is unknown again.",
    );
    await expect(voices).toContainText("No confirmed voice you can see.");
    await expect(claire).toContainText("0 voice(s)");
    // With no voice left, the action stays visible and says why.
    await expect(separate).toHaveAttribute("aria-disabled", "true");
    await expect(separate).toHaveAccessibleDescription(
      "Claire Synthétique has no confirmed voice to separate.",
    );

    // The corrections are kept in the log, on the directory.
    await page.getByRole("link", { name: "Back to people" }).click();
    await page.getByRole("button", { name: "Corrections log" }).click();
    const log = page.getByRole("list", { name: "Person corrections" });
    await expect(log).toContainText(
      "Merge: Alicia Synthétique → Claire Synthétique",
    );
    await expect(log).toContainText("Separation: Claire Synthétique");

    // The separated voice is back in the verification.
    await page.goto(`/people/verification?clip=${clipId}`);
    await expect(
      page
        .getByRole("region", { name: "Who is speaking here?" })
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
    await expect(deletion).toContainText("0 voice(s) confirmed for Claire");
    await expect(deletion).toContainText("0 validated sample(s) leave");
    await expect(deletion).toContainText("0 microphone track rule(s) naming");
    await expect(deletion).toContainText("The clips, the voice fingerprints");
    await expect(deletion.getByRole("textbox")).toHaveCount(0);
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

  test("actions bloquées : raison écrite, conséquences chiffrées et refus du serveur", async ({
    browser,
  }) => {
    test.setTimeout(120_000);
    const [linked, target] = await createPersons([
      "Liée Synthétique",
      "Cible Synthétique",
    ]);
    await ensureMember();
    await linkAccount(linked, MEMBER.username);
    const record = (name: string) => page.getByRole("region", { name });

    // A Person linked to an account: merge and deletion stay visible, disabled, with the reason.
    await page.goto(`/people/${linked}`);
    const lockedRecord = record("Liée Synthétique");
    const merge = lockedRecord.getByRole("button", {
      name: "Merge Liée Synthétique",
    });
    const remove = lockedRecord.getByRole("button", {
      name: "Delete Liée Synthétique",
    });
    await expect(merge).toHaveAttribute("aria-disabled", "true");
    await expect(merge).toHaveAccessibleDescription(
      "Linked to an account, Liée Synthétique cannot be merged away. Open the other person and merge it into Liée Synthétique instead.",
    );
    await expect(remove).toHaveAttribute("aria-disabled", "true");
    await expect(remove).toHaveAccessibleDescription(
      "Linked to an account, Liée Synthétique cannot be deleted here. Delete the voice profile from the account instead.",
    );
    // Reachable by keyboard, and activating it opens nothing.
    await merge.focus();
    await expect(merge).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page.getByRole("alertdialog")).toHaveCount(0);
    // The actions are in the header, never in the list.
    await expect(
      page
        .getByRole("list", { name: "People", exact: true })
        .getByRole("button"),
    ).toHaveCount(0);
    await expectNoViolations(page);

    // A voice on a clip that starts shared and becomes private under the feet of a member.
    const clip = await uploadClip("tour-prive.mp4", "shared");
    const [first] = await transcribeAndDiarize(clip, {
      secondFingerprintEnd: 4.2,
    });
    await nameVoice(first, "Voix Bascule");
    const admin = await adminApi();
    const found = (await (
      await admin.api.get("/api/persons?q=Voix%20Bascule")
    ).json()) as { items: { id: number }[] };
    const bascule = found.items[0].id;

    const context = await browser.newContext();
    await context.addCookies((await sessionFor(MEMBER)).cookies);
    const member = await context.newPage();
    await member.goto(`/people/${bascule}`);
    const memberRecord = member.getByRole("region", { name: "Voix Bascule" });
    await expect(
      memberRecord.getByRole("button", { name: "Delete Voix Bascule" }),
    ).toHaveAttribute("aria-disabled", "false");

    // The clip becomes private after the record was loaded: the server refuses (409).
    const me = await (await admin.api.get("/api/auth/me")).json();
    const patched = await admin.api.patch(`/api/clips/${clip}`, {
      data: { visibility: "private" },
      headers: {
        Origin: new URL(member.url()).origin,
        "X-CSRF-Token": me.csrf_token,
      },
    });
    expect(patched.ok()).toBe(true);
    await memberRecord
      .getByRole("button", { name: "Delete Voix Bascule" })
      .click();
    const deletion = member.getByRole("alertdialog", {
      name: "Delete Voix Bascule?",
    });
    await expect(deletion).toContainText("1 voice(s) confirmed for Voix");
    await expect(deletion).toContainText("1 validated sample(s) leave");
    await deletion.getByRole("button", { name: "Delete", exact: true }).click();
    // A precise message, the record stays open, and it now explains the refusal.
    await expect(member.getByRole("alert")).toContainText(
      "Nothing was changed. This person still has hidden voices.",
    );
    await expect(memberRecord).toBeVisible();
    await expect(member).toHaveURL(new RegExp(`/people/${bascule}$`));
    const blocked = memberRecord.getByRole("button", {
      name: "Delete Voix Bascule",
    });
    await expect(blocked).toHaveAttribute("aria-disabled", "true");
    await expect(blocked).toHaveAccessibleDescription(
      "Some voices of Voix Bascule are on clips you cannot see: only someone who can see them can delete Voix Bascule.",
    );
    await expect(
      memberRecord.getByRole("button", { name: "Merge Voix Bascule" }),
    ).toHaveAccessibleDescription(
      "Some voices of Voix Bascule are on clips you cannot see: only someone who can see them can merge Voix Bascule.",
    );
    // Its confirmed voices are hidden from the member: the separation says so.
    await expect(
      memberRecord.getByRole("button", {
        name: "Separate voices of Voix Bascule",
      }),
    ).toHaveAccessibleDescription(
      "The confirmed voices of Voix Bascule are on clips you cannot see.",
    );
    await expectNoViolations(member);
    await context.close();

    // The administrator sees every clip: the same Person can be merged, with its counts.
    await page.goto(`/people/${bascule}`);
    await record("Voix Bascule")
      .getByRole("button", { name: "Merge Voix Bascule" })
      .click();
    const dialog = page.getByRole("dialog", { name: "Merge Voix Bascule" });
    await dialog.getByRole("searchbox", { name: "Merge into" }).fill("Cible");
    await dialog.getByRole("radio", { name: "Cible Synthétique" }).check();
    await expect(dialog).toContainText(
      "1 confirmed voice(s) of Voix Bascule are now confirmed as Cible Synthétique.",
    );
    await expect(dialog).toContainText(
      "1 validated sample(s) of Voix Bascule become references of Cible Synthétique.",
    );
    await dialog.getByRole("button", { name: "Merge", exact: true }).click();
    await expect(page).toHaveURL(new RegExp(`/people/${target}$`));
    await expect(record("Cible Synthétique")).toContainText("1 voice(s)");

    // Back to the instance the next journeys expect: no reference and no extra clip.
    const headers = {
      Origin: new URL(page.url()).origin,
      "X-CSRF-Token": me.csrf_token,
    };
    for (const path of [`/api/persons/${target}`, `/api/admin/clips/${clip}`]) {
      expect((await admin.api.delete(path, { headers })).ok()).toBe(true);
    }
    await admin.api.dispose();
  });

  test("personnes et vérification en thème sombre et à 390 pixels", async () => {
    await page.evaluate(() => localStorage.setItem("snitch-ui-theme", "dark"));
    await page.setViewportSize({ width: 390, height: 780 });
    for (const path of [
      "/people",
      "/people/verification",
      `/people/verification?clip=${clipId}`,
    ]) {
      await page.goto(path);
      await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
      await expectNoViolations(page);
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= window.innerWidth,
        ),
      ).toBe(true);
    }
    const identify = page
      .getByRole("region", { name: "Who is speaking here?" })
      .getByRole("button", { name: "Identify" });
    await expect(identify).toBeVisible();
    expect((await identify.boundingBox())!.height).toBeGreaterThanOrEqual(44);
    await expectNoViolations(page);

    // The record with its actions and written reasons, on a phone, in the dark.
    await page.goto("/people");
    await page
      .getByRole("list", { name: "People", exact: true })
      .getByRole("link", { name: /Liée Synthétique/ })
      .click();
    const merge = page.getByRole("button", { name: "Merge Liée Synthétique" });
    await expect(merge).toBeVisible();
    await expect(merge).toHaveAccessibleDescription(/Linked to an account/);
    for (const button of await page
      .getByRole("group", { name: "Actions on Liée Synthétique" })
      .getByRole("button")
      .all()) {
      expect((await button.boundingBox())!.height).toBeGreaterThanOrEqual(44);
    }
    await expectNoViolations(page);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
  });

  test("la légende des badges reste lisible en thème sombre à 390 pixels", async () => {
    await page.goto(`/people/verification?clip=${clipId}`);
    const caseBadge = page
      .getByRole("region", { name: "Who is speaking here?" })
      .getByRole("button", { name: "Unknown" });
    await caseBadge.click();
    const legend = page.getByRole("dialog", { name: "Unknown" });
    await expect(legend).toBeVisible();
    await expect(legend).toContainText("Counts as a Reference: no");
    const box = (await legend.boundingBox())!;
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(390);
    // The legend is above the fixed navigation and the sticky bars: the point at the
    // middle of its visible part belongs to it.
    const onTop = await legend.evaluate((node) => {
      const r = node.getBoundingClientRect();
      const y = Math.min(r.bottom, window.innerHeight) - 4;
      const hit = document.elementFromPoint(r.left + r.width / 2, y);
      return hit !== null && node.contains(hit);
    });
    expect(onTop).toBe(true);
    await expectNoViolations(page);
    // It does not stay on the content: Escape closes it.
    await page.keyboard.press("Escape");
    await expect(legend).toBeHidden();
  });

  test("vérification : recherche serveur, enchaînement, Entrée au clavier et cas déjà tranché", async () => {
    // Twelve more Persons than a page of results holds, and a second clip with new voices.
    const names = Array.from(
      { length: 12 },
      (_, i) => `Zoé Synthétique ${String(i + 1).padStart(2, "0")}`,
    );
    const ids = await createPersons(names);
    const second = await uploadClip("second-voices.mp4", "shared");
    await transcribeAndDiarize(second, { secondFingerprintEnd: 4.2 });
    // Two validated references, each close to both voices of the second clip, from a third
    // clip: those voices become ambiguous and the server proposes both Persons as
    // candidates, so the number keys below always have something to choose.
    const third = await uploadClip("third-voices.mp4", "shared");
    const thirdVoices = await transcribeAndDiarize(third, {
      secondFingerprintEnd: 4.2,
    });
    await nameVoice(thirdVoices[0], "Référence Synthétique A");
    await nameVoice(thirdVoices[1], "Référence Synthétique B");
    await page.evaluate(() => localStorage.setItem("snitch-ui-theme", "light"));
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto(`/people/verification?clip=${second}`);

    const current = page.getByRole("region", { name: "Who is speaking here?" });
    const heading = current.getByRole("heading", { level: 2 });
    const tab = page.getByRole("link", { name: /^To process\s*\d+$/ });
    await expect(heading).toBeVisible();
    const count = async () =>
      Number((await tab.innerText()).replace(/\D/g, ""));
    await expect.poll(count).toBeGreaterThanOrEqual(2);
    const before = await count();
    const speakerOf = async () =>
      Number(
        (await current.locator("audio").getAttribute("src"))!.match(
          /local-speakers\/(\d+)\//,
        )![1],
      );

    // The closest people come with their proximity, never as a confirmation.
    const candidates = current.getByRole("list", { name: "Closest people" });
    await expect(candidates).toContainText("% proximity");
    await expect(
      candidates.getByRole("button", { name: /^It is Référence Synthétique/ }),
    ).not.toHaveCount(0);

    // The search is made by the server: the eleventh person is found past the first page.
    await current.getByRole("button", { name: "Identify" }).click();
    await current
      .getByRole("searchbox", { name: "Search people" })
      .fill("Synthétique 11");
    const first = await speakerOf();
    await current
      .getByRole("button", { name: "Identify as Zoé Synthétique 11" })
      .click();
    // The next voice follows by itself and the counter of the tab goes down.
    await expect.poll(speakerOf).not.toBe(first);
    await expect.poll(count).toBeLessThan(before);

    // Somebody else settles the displayed voice first: the choice is refused, nothing is
    // written for the chosen Person and the message names who won.
    const target = await speakerOf();
    const { api, post } = await adminApi();
    await post(`/api/local-speakers/${target}/validation`, {
      person_id: ids[0],
    });
    await current.getByRole("button", { name: "Identify" }).click();
    await current
      .getByRole("searchbox", { name: "Search people" })
      .fill("Synthétique 12");
    await current
      .getByRole("button", { name: "Identify as Zoé Synthétique 12" })
      .click();
    await expect(page.getByRole("alert")).toContainText(
      "Already confirmed as Zoé Synthétique 01.",
    );
    const twelve = (await (
      await api.get(`/api/persons/${ids[11]}`)
    ).json()) as { local_speaker_count: number };
    expect(twelve.local_speaker_count).toBe(0);
    await api.dispose();

    // Reading the screen again: both voices of the clip are settled, the counter agrees.
    await page.reload();
    await expect(
      page.getByText("All the voices of this clip are processed."),
    ).toBeVisible();
    await expect.poll(count).toBe(before - 2);
    await expectNoViolations(page);
  });

  test("liens directs : une voix inexistante et une voix invisible disent la même chose", async ({
    browser,
  }) => {
    // A private clip of the administrator, with its voices, and a member without access.
    const hiddenClip = await uploadClip("private-secret.mp4", "private");
    await transcribeAndDiarize(hiddenClip, {});
    const { api } = await adminApi();
    const voice = (await (
      await api.get(`/api/clips/${hiddenClip}/voice`)
    ).json()) as { speakers: { id: number }[] };
    await api.dispose();
    await ensureMember();
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const member = await context.newPage();
    const messages: string[] = [];
    for (const id of [voice.speakers[0].id, 999999]) {
      await member.goto(`/people/verification?case=${id}`);
      const notice = member.getByRole("region", {
        name: "Direct link to a voice",
      });
      await expect(notice.getByRole("heading", { level: 2 })).toHaveText(
        "Voice not found or not accessible",
      );
      messages.push((await notice.textContent()) ?? "");
      await expect(member.getByText("private-secret.mp4")).toHaveCount(0);
      // The queue may go on below the notice, but never with the hidden voice.
      await expect(
        member.locator(`audio[src*="/local-speakers/${id}/"]`),
      ).toHaveCount(0);
    }
    expect(messages[0]).toBe(messages[1]);
    await expectNoViolations(member);
    // The same message in the dark theme, on a phone.
    await member.addInitScript(() =>
      localStorage.setItem("snitch-ui-theme", "dark"),
    );
    await member.setViewportSize({ width: 390, height: 780 });
    await member.goto("/people/verification?case=999999");
    await expect(
      member.getByRole("button", { name: "Continue the review" }),
    ).toBeVisible();
    const box = await member
      .getByRole("button", { name: "Continue the review" })
      .boundingBox();
    expect(box!.height).toBeGreaterThanOrEqual(44);
    expect(
      await member.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    await expectNoViolations(member);
    await context.close();
  });
});
