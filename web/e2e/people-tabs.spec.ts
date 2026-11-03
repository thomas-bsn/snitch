import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// The People structure: three tabs (Directory, To process, Unknown voices) and the two
// addresses that open a case or a voice group. The clip-processing screen and the group panel
// come later; until then their addresses keep the query and land on the right section.

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

test.describe.serial("structure de Personnes", () => {
  test.beforeAll(async ({ browser }) => {
    page = await (await browser.newContext()).newPage();
    await prepareDiarizedClip(page, { secondFingerprintEnd: 4.2 });
  });
  test.afterAll(async () => {
    await page.context().close();
  });

  test("trois onglets, Annuaire par défaut", async () => {
    await page.goto("/people");
    const tabs = page.getByRole("navigation", { name: "People sections" });
    await expect(tabs.getByRole("link")).toHaveText([
      "Directory",
      "To process",
      "Unknown voices",
    ]);
    await expect(tabs.getByRole("link", { name: "Directory" })).toHaveAttribute(
      "aria-current",
      "page",
    );
    await expectNoViolations(page);
  });

  test("l'onglet To process mène à la file de vérification", async () => {
    await page
      .getByRole("navigation", { name: "People sections" })
      .getByRole("link", { name: "To process" })
      .click();
    await expect(page).toHaveURL(/\/people\/verification$/);
    await expect(
      page.getByRole("list", { name: "Voices to decide" }),
    ).toBeVisible();
  });

  test("l'adresse de Cas de vérification garde son paramètre", async () => {
    await page.goto("/people/verification?case=42");
    await expect(page).toHaveURL(/\/people\/verification\?case=42$/);
    await expect(page.getByTestId("people-case-placeholder")).toContainText(
      "42",
    );
    await expect(
      page
        .getByRole("navigation", { name: "People sections" })
        .getByRole("link", { name: "To process" }),
    ).toHaveAttribute("aria-current", "page");
  });

  test("l'adresse de voix garde son paramètre et mène aux voix inconnues", async () => {
    await page.goto("/people/groups?voice=7");
    await expect(page).toHaveURL(/\/people\/groups\?voice=7$/);
    await expect(page.getByTestId("people-voice-placeholder")).toContainText(
      "7",
    );
    await expect(
      page
        .getByRole("navigation", { name: "People sections" })
        .getByRole("link", { name: "Unknown voices" }),
    ).toHaveAttribute("aria-current", "page");
  });
});
