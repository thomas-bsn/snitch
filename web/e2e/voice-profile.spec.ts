import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Browser, type Page } from "@playwright/test";
import {
  adminApi,
  nameVoice,
  transcribeAndDiarize,
  uploadClip,
} from "./library-fixture";
import { createPersons, linkAccount, personName } from "./people-fixture";
import {
  ADMIN,
  createApprovedMember,
  installAdministrator,
  MEMBER,
  sessionFor,
  setAccountLanguage,
} from "./shell-fixture";

// "My voice": the four states of the link request (no Person, pending, refused, linked),
// the server-side search of a Person, and the sensitive zone (delete my voice profile,
// unlink my account) with the exact numbers of what is erased and what stays. The data is
// synthetic: a diarized clip with one named voice and a directory of generated Persons.

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];
const OWN = "Camille Synthétique";
const GENERATED = 60;

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

let ownId: number;
let clipId: number;
let impact: {
  voice_samples: number;
  fingerprints: number;
  attributions: number;
};

async function memberPage(
  browser: Browser,
  options: {
    viewport?: { width: number; height: number };
    theme?: string;
  } = {},
) {
  const context = await browser.newContext({
    storageState: await sessionFor(MEMBER),
    viewport: options.viewport,
  });
  if (options.theme) {
    await context.addInitScript((value) => {
      localStorage.setItem("snitch-ui-theme", value);
    }, options.theme);
  }
  const page = await context.newPage();
  await page.goto("/account/voice");
  return { context, page };
}

async function decideLatestRequest(decision: "approve" | "refuse") {
  const { api, post } = await adminApi();
  const queue = (await (
    await api.get("/api/admin/person-link-requests")
  ).json()) as { items: { id: number }[] };
  await post(
    `/api/admin/person-link-requests/${queue.items[0].id}/${decision}`,
  );
  await api.dispose();
}

async function readImpact() {
  const { api } = await adminApi();
  const body = await (await api.get(`/api/persons/${ownId}/impact`)).json();
  await api.dispose();
  return body as typeof impact;
}

async function requestLink(page: Page, search: string, note: string) {
  await page
    .getByRole("searchbox", { name: "Search for a person" })
    .fill(search);
  await page.getByRole("button", { name: OWN }).click();
  await page.getByLabel("Message (optional)").fill(note);
  await page.getByRole("button", { name: "Request the link" }).click();
}

test.describe.serial("ma voix", () => {
  test.beforeAll(async () => {
    test.setTimeout(120_000);
    await installAdministrator();
    await createApprovedMember();
    await createPersons(
      Array.from({ length: GENERATED }, (_, i) => personName(i + 1)),
    );
    clipId = await uploadClip("synthetic-voices.mp4", "shared");
    const speakers = await transcribeAndDiarize(clipId);
    await nameVoice(speakers[0], OWN);
    const { api } = await adminApi();
    const found = (await (
      await api.get(`/api/persons?q=${encodeURIComponent("Camille")}`)
    ).json()) as { items: { id: number }[] };
    ownId = found.items[0].id;
    await api.dispose();
  });

  test("aucune Personne : la recherche interroge le serveur et dit comment affiner", async ({
    browser,
  }) => {
    const { context, page } = await memberPage(browser);
    const card = page.getByRole("main");
    await expect(card).toContainText("No person is linked to your account");
    await expect(
      page.getByRole("button", { name: "Delete my voice profile" }),
    ).toHaveCount(0);
    await page
      .getByRole("searchbox", { name: "Search for a person" })
      .fill("Personne");
    await expect(
      card.getByText("Showing 8 of 60 people. Type more of the name"),
    ).toBeVisible();
    expect(
      await card.getByRole("button", { name: /^Personne \d{4}$/ }).count(),
    ).toBe(8);
    await page
      .getByRole("searchbox", { name: "Search for a person" })
      .fill("zzzz");
    await expect(card).toContainText("No person matches this search.");
    await context.close();
  });

  test("demande en attente : progression en trois étapes, annulation, nouvelle demande", async ({
    browser,
  }) => {
    const { context, page } = await memberPage(browser);
    await requestLink(page, "Camille", "C'est ma voix.");
    const steps = page.getByRole("list", { name: "Progress of your request" });
    await expect(steps.getByRole("listitem")).toHaveCount(3);
    await expect(steps.getByRole("listitem").nth(0)).toContainText(
      "Request sent",
    );
    await expect(steps.getByRole("listitem").nth(1)).toContainText(
      "Validation by an administrator",
    );
    await expect(steps.getByRole("listitem").nth(1)).toHaveAttribute(
      "aria-current",
      "step",
    );
    await expect(steps.getByRole("listitem").nth(2)).toContainText(
      "Voice identified",
    );
    await expect(page.getByRole("main")).toContainText(OWN);
    await page.getByRole("button", { name: "Cancel the request" }).click();
    await expect(page.getByRole("status")).toContainText("Request cancelled.");
    await expect(
      page.getByRole("searchbox", { name: "Search for a person" }),
    ).toBeVisible();
    await context.close();
  });

  test("demande refusée : message explicite et nouvelle demande possible", async ({
    browser,
  }) => {
    const { context, page } = await memberPage(browser);
    await requestLink(page, "Camille", "Première demande.");
    await expect(
      page.getByRole("list", { name: "Progress of your request" }),
    ).toBeVisible();
    await decideLatestRequest("refuse");
    await page.reload();
    await expect(page.getByRole("main")).toContainText(
      `Your request for ${OWN} was refused by an administrator`,
    );
    await expect(
      page.getByRole("list", { name: "Progress of your request" }),
    ).toHaveCount(0);
    await requestLink(page, "Camille", "Seconde demande.");
    await expect(
      page.getByRole("list", { name: "Progress of your request" }),
    ).toBeVisible();
    await context.close();
  });

  test("Personne liée : compteurs, accès à mes clips, accessibilité", async ({
    browser,
  }) => {
    await decideLatestRequest("approve");
    for (const theme of ["light", "dark"]) {
      const { context, page } = await memberPage(browser, { theme });
      const main = page.getByRole("main");
      await expect(main).toContainText(`You are the person ${OWN}.`);
      const counters = main.getByRole("list", { name: "My voice in numbers" });
      await expect(counters).toContainText("Validated voice samples");
      await expect(counters).toContainText("Voices");
      await expect(
        counters.getByRole("listitem").filter({ hasText: "Clips" }),
      ).toContainText("1");
      await expect(
        main.getByRole("link", { name: "See my clips" }),
      ).toHaveAttribute("href", `/library?person=${ownId}`);
      await expectNoViolations(page);
      await context.close();
    }
    const phone = await memberPage(browser, {
      viewport: { width: 390, height: 780 },
    });
    expect(
      await phone.page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    for (const button of await phone.page.getByRole("button").all()) {
      expect((await button.boundingBox())!.height).toBeGreaterThanOrEqual(44);
    }
    await phone.context.close();
  });

  test("délier : ce qui est perdu et ce qui reste, sans rien supprimer", async ({
    browser,
  }) => {
    const { context, page } = await memberPage(browser);
    await expect(page.getByRole("main")).toContainText(
      "The person, its voices and its clips are not deleted",
    );
    await page
      .getByRole("button", { name: "Unlink my account from this person" })
      .click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toContainText("What you lose");
    await expect(dialog).toContainText("What stays");
    await dialog.getByRole("button", { name: "Unlink", exact: true }).click();
    await expect(page.getByRole("status")).toContainText("Account unlinked.");
    await expect(page.getByRole("main")).toContainText(
      "No person is linked to your account",
    );
    // Person, voice and clip are still there.
    const { api } = await adminApi();
    const person = await (await api.get(`/api/persons/${ownId}`)).json();
    expect(person.user_id).toBeNull();
    expect(person.local_speaker_count).toBeGreaterThanOrEqual(1);
    expect((await api.get(`/api/clips/${clipId}`)).ok()).toBe(true);
    await api.dispose();
    await context.close();
  });

  test("supprimer mon profil vocal : nombres exacts, les clips restent", async ({
    browser,
  }) => {
    await linkAccount(ownId, MEMBER.username);
    impact = await readImpact();
    expect(impact.voice_samples).toBeGreaterThanOrEqual(1);
    const { context, page } = await memberPage(browser);
    const effects = `${impact.voice_samples} validated voice sample(s), ${impact.fingerprints} voice fingerprint(s) and ${impact.attributions} attribution(s)`;
    await expect(page.getByRole("main")).toContainText(effects);
    await expect(page.getByRole("main")).toContainText(
      "Your clips stay in place",
    );
    await page.getByRole("button", { name: "Delete my voice profile" }).click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toContainText(effects);
    await expect(dialog).toContainText("Your clips stay in place");
    await dialog
      .getByRole("button", { name: "Delete my voice profile" })
      .click();
    await expect(page.getByRole("status")).toContainText(
      `Voice profile deleted: ${impact.voice_samples} sample(s), ${impact.fingerprints} fingerprint(s) and ${impact.attributions} attribution(s) erased.`,
    );
    await expect(
      page.getByRole("button", { name: "Delete my voice profile" }),
    ).toHaveCount(0);
    expect(await readImpact()).toMatchObject({
      voice_samples: 0,
      fingerprints: 0,
      attributions: 0,
    });
    await context.close();
  });

  test("en français : l'état supprimé est dit dans la langue du compte", async ({
    browser,
  }) => {
    await setAccountLanguage(MEMBER, "fr");
    const { context, page } = await memberPage(browser);
    await expect(page.getByRole("main")).toContainText(
      `Vous êtes la Personne ${OWN}.`,
    );
    await expect(page.getByRole("main")).toContainText(
      "Votre profil vocal est supprimé",
    );
    await expectNoViolations(page);
    await context.close();
    await setAccountLanguage(MEMBER, "en");
  });

  test("administrateur sans lien : ses états restent cohérents", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/account/voice");
    await expect(page.getByRole("main")).toContainText(
      "No person is linked to your account",
    );
    await context.close();
  });
});
