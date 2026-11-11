import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import {
  durationText,
  videoCountText,
} from "../src/app/people/referenceFigures";
import { createPersons, linkAccount, personName } from "./people-fixture";
import {
  ADMIN,
  createApprovedMember,
  installAdministrator,
  MEMBER,
  sessionFor,
} from "./shell-fixture";

// The list of Persons and its record, on a synthetic directory of 2002 Persons: the search
// and the paging are done by the server, and a Person has an address of its own. The
// instance is installed once, so the journeys share it and follow one another.

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];
const COUNT = 2000;
const SPECIALS = ["Élodie Synthétique", "Cent%_pour cent"];
const TOTAL = COUNT + SPECIALS.length;
const PAGE_SIZE = 25;
const PAGES = Math.ceil(TOTAL / PAGE_SIZE);

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

// The number of videos of a card: agreed in number, written for the language, compact from 10,000, the exact
// number kept for the accessible name and the tooltip. U+202F and U+00A0 are the French spaces.
test("nombre de vidéos d'une carte aux bornes", () => {
  const cases: [number, "en" | "fr", string, string][] = [
    [1, "en", "1 video", "1 video"],
    [1, "fr", "1 vidéo", "1 vidéo"],
    [8, "fr", "8 vidéos", "8 vidéos"],
    [9999, "en", "9,999 videos", "9,999 videos"],
    [9999, "fr", "9 999 vidéos", "9 999 vidéos"],
    [12000, "en", "12K videos", "12,000 videos"],
    [12000, "fr", "12 k vidéos", "12 000 vidéos"],
  ];
  for (const [count, language, short, exact] of cases) {
    expect(videoCountText(count, language), `${count} ${language}`).toEqual({
      short,
      exact,
    });
  }
});

// A duration is never rounded down to "0 min": seconds under a minute, minutes and seconds, then hours.
test("durée lisible de Parole nette", () => {
  const cases: [number, string][] = [
    [0, "0 s"],
    [0.4, "< 1 s"],
    [12.3, "12 s"],
    [48, "48 s"],
    [59.6, "1 min"],
    [192, "3 min 12 s"],
    [3900, "1 h 05"],
    [36000, "10 h 00"],
  ];
  for (const [seconds, text] of cases) {
    expect(durationText(seconds), String(seconds)).toBe(text);
  }
});

let page: Page;
let accountPersonId: number;
const list = () => page.getByRole("list", { name: "People", exact: true });
const search = () => page.getByRole("searchbox", { name: "Search people" });

test.describe.serial("liste des personnes", () => {
  test.beforeAll(async ({ browser }) => {
    test.setTimeout(180_000);
    await installAdministrator();
    const ids = await createPersons([
      ...Array.from({ length: COUNT }, (_, i) => personName(i + 1)),
      ...SPECIALS,
    ]);
    accountPersonId = ids[41]; // "Personne 0042"
    await createApprovedMember();
    await linkAccount(accountPersonId, MEMBER.username);
    page = await (await browser.newContext()).newPage();
    await page.context().addCookies((await sessionFor(ADMIN)).cookies);
  });
  test.afterAll(async () => {
    await page.context().close();
  });

  test("le total vient du serveur et la liste se pagine", async () => {
    await page.goto("/people");
    await expect(
      page.getByText(`Page 1 of ${PAGES} (${TOTAL} person(s))`),
    ).toBeVisible();
    await expect(list().getByRole("listitem")).toHaveCount(PAGE_SIZE);
    const first = await list().getByRole("listitem").first().innerText();

    await page.getByRole("button", { name: "Next page" }).click();
    await expect(page.getByText(`Page 2 of ${PAGES}`)).toBeVisible();
    await expect(list().getByRole("listitem")).toHaveCount(PAGE_SIZE);
    await expect(list().getByRole("listitem").first()).not.toHaveText(first);
    await page.getByRole("button", { name: "Previous page" }).click();
    await expect(page.getByText(`Page 1 of ${PAGES}`)).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Previous page" }),
    ).toBeDisabled();

    // The last page holds the remainder.
    for (let n = 1; n < PAGES; n += 1) {
      await page.getByRole("button", { name: "Next page" }).click();
    }
    await expect(page.getByText(`Page ${PAGES} of ${PAGES}`)).toBeVisible();
    await expect(list().getByRole("listitem")).toHaveCount(
      TOTAL - PAGE_SIZE * (PAGES - 1),
    );
    await expect(
      page.getByRole("button", { name: "Next page" }),
    ).toBeDisabled();
  });

  test("la recherche est faite par le serveur et se combine avec la page", async () => {
    await page.goto("/people");
    await search().fill("Personne 07");
    // 0700 to 0799: one hundred people, four pages.
    await expect(page.getByText("Page 1 of 4 (100 person(s))")).toBeVisible();
    await expect(
      list().getByRole("listitem").first().getByRole("link"),
    ).toContainText("Personne 0700");
    await page.getByRole("button", { name: "Next page" }).click();
    await expect(
      list().getByRole("listitem").first().getByRole("link"),
    ).toContainText("Personne 0725");

    // A new search starts again at the first page.
    await search().fill("Personne 1999");
    await expect(page.getByText("Page 1 of 1 (1 person(s))")).toBeVisible();
    await expect(list().getByRole("listitem")).toHaveCount(1);

    // The wildcards of the search stay literal.
    await search().fill("%_");
    await expect(list().getByRole("listitem")).toHaveCount(1);
    await expect(list()).toContainText("Cent%_pour cent");

    await search().fill("introuvable");
    await expect(
      page.getByText("No person matches this search."),
    ).toBeVisible();
    await expect(list()).toHaveCount(0);
  });

  test("la liste montre les chiffres réels et le compte lié", async () => {
    await page.goto("/people");
    await search().fill("Personne 0042");
    const item = list().getByRole("listitem");
    await expect(item).toHaveCount(1);
    await expect(item).toContainText("Account");
    // The card shows the number of videos, agreed in number, and no speaking time.
    await expect(item).toContainText("0 videos");
    await expect(item).not.toContainText("min");
    await expect(item).toContainText("Reference: none");
    await expect(
      item.getByTitle("No sample usable by recognition yet"),
    ).toHaveText("Reference: none");
    // The list holds no action.
    await expect(item.getByRole("button")).toHaveCount(0);

    await search().fill("Personne 0043");
    await expect(list().getByRole("listitem")).toContainText("Reference: none");
    await expect(list().getByRole("listitem")).not.toContainText("Account");
  });

  test("la fiche a une adresse partageable et un identifiant inconnu est introuvable", async () => {
    await page.goto("/people");
    await search().fill("Personne 0042");
    await list()
      .getByRole("link", { name: /Personne 0042/ })
      .click();
    await expect(page).toHaveURL(new RegExp(`/people/${accountPersonId}$`));
    const record = page.getByRole("region", { name: "Personne 0042" });
    await expect(
      record.getByRole("heading", { level: 2, name: "Personne 0042" }),
    ).toBeVisible();
    // The four figures of the record: no sample, hence no Reference; the linked account by name.
    await expect(record).toContainText("Reference samples0");
    await expect(record).toContainText("Reference qualitynone");
    // The Reference as recognition uses it, with the same level as the card.
    const reference = record.getByRole("group", {
      name: "Reference of Personne 0042",
    });
    await expect(reference).toContainText("Samples used0 of 0 validated");
    await expect(reference).toContainText("Clean speech0 s");
    await expect(reference).toContainText("Reference: none");
    await expect(record).toContainText(`Linked account: @${MEMBER.username}`);
    // The sheet is a full page: the grid is gone, and a weak Reference is flagged.
    await expect(list()).toHaveCount(0);
    await expect(record.getByRole("note")).toContainText(
      "Reference to strengthen",
    );
    await expect(
      record.getByRole("group", {
        name: "Videos where Personne 0042 is heard",
      }),
    ).toContainText("No video you can see.");
    await expect(
      page
        .getByRole("navigation", { name: "People sections" })
        .getByRole("link", { name: /^Directory\s*\d*$/ }),
    ).toHaveAttribute("aria-current", "page");

    // The address works on its own, in a fresh page.
    await page.goto(`/people/${accountPersonId}`);
    await expect(
      page.getByRole("heading", { level: 2, name: "Personne 0042" }),
    ).toBeVisible();
    await expectNoViolations(page);

    for (const path of ["/people/999999", "/people/not-a-number"]) {
      await page.goto(path);
      await expect(page.getByText("Person not found")).toBeVisible();
      await page.getByRole("link", { name: "Back to people" }).click();
      await expect(page).toHaveURL(/\/people$/);
    }
  });

  test("thème sombre et 390 pixels : deux écrans successifs avec retour", async () => {
    await page.evaluate(() => localStorage.setItem("snitch-ui-theme", "dark"));
    await page.setViewportSize({ width: 390, height: 780 });
    await page.goto("/people");
    await expect(list()).toBeVisible();
    // On a phone the list and the record are two screens.
    await expect(page.getByRole("heading", { level: 2 })).toHaveCount(0);
    await expectNoViolations(page);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);

    await list().getByRole("link").first().click();
    await expect(page.getByRole("heading", { level: 2 })).toBeVisible();
    await expect(list()).toHaveCount(0);
    await expectNoViolations(page);
    await page.getByRole("link", { name: "Back to people" }).click();
    await expect(list()).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
  });

  test("la liste et la fiche en français", async () => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.evaluate(() => localStorage.setItem("snitch-ui-language", "fr"));
    await page.goto(`/people/${accountPersonId}`);
    await expect(
      page.getByRole("heading", { level: 2, name: "Personne 0042" }),
    ).toBeVisible();
    await expect(page.getByText("Échantillons de Référence0")).toBeVisible();
    await expect(
      page.getByText(`Compte lié : @${MEMBER.username}`),
    ).toBeVisible();
    await expect(page.getByText("Référence à renforcer")).toBeVisible();
    await page.goto("/people");
    await expect(page.getByText(`Page 1 sur ${PAGES}`)).toBeVisible();
    await expect(page.getByText("Référence : aucune").first()).toBeVisible();
    await expect(
      page.getByText("0 vidéo", { exact: true }).first(),
    ).toBeVisible();
    await page.goto(`/people/${accountPersonId}`);
    await expectNoViolations(page);
    await page.goto("/people/999999");
    await expect(page.getByText("Personne introuvable")).toBeVisible();
  });
});
