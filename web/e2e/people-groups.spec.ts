import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// The Voice groups section of the People screen: close unknown voices proposed together,
// the detail with one checkbox per voice, leave a voice out and put it back, the sticky bar
// that names what is kept, the conflict state and the address of a voice. The two close synthetic voices of the shared fixture form the
// group; the instance is installed through the guided setup.

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

/** A change made behind the panel's back, as another user would. */
async function changeGroup(
  page: Page,
  method: "PUT" | "DELETE",
  voice: "SPEAKER_00" | "SPEAKER_01",
) {
  await page.evaluate(
    async ([verb, label]) => {
      const me = await fetch("/api/auth/me").then((r) => r.json());
      const groups = await fetch(
        "/api/voice-groups?proposed=false&page_size=100",
      ).then((r) => r.json());
      const member = groups.items
        .flatMap(
          (g: { members: { label: string; local_speaker_id: number }[] }) =>
            g.members,
        )
        .find((m: { label: string }) => m.label === label);
      await fetch(
        `/api/local-speakers/${member.local_speaker_id}/group-exclusion`,
        {
          method: verb,
          headers: { "X-CSRF-Token": me.csrf_token, Origin: location.origin },
        },
      );
    },
    [method, voice],
  );
}

let page: Page;

test.describe.serial("groupes de voix", () => {
  test.beforeAll(async ({ browser }) => {
    page = await (await browser.newContext()).newPage();
    await prepareDiarizedClip(page, { secondFingerprintEnd: 4.2 });
  });
  test.afterAll(async () => {
    await page.context().close();
  });

  test("un groupe modifié entre-temps est refusé sans rien écrire, puis rechargé", async () => {
    await page.goto("/people/groups");
    await expect(page.getByRole("heading", { level: 1 })).toHaveText("People");
    await expect(
      page.getByRole("link", { name: /^Unknown voices/ }),
    ).toHaveAttribute("aria-current", "page");
    await expect(page.getByText("no group is ever confirmed")).toBeVisible();

    const groups = page.getByRole("list", { name: "Groups of unknown voices" });
    const card = groups.locator("xpath=./li");
    await expect(card).toHaveCount(1);
    await expect(card).toContainText("Proposed grouping of 2 voices");
    await expect(card).toContainText(
      "Heard in 1 clip(s): synthetic-voices.mp4",
    );
    await expect(card).not.toContainText("dentified");
    await expectNoViolations(page);

    await card.getByRole("button", { name: /^Proposed grouping of/ }).click();
    await expect(page).toHaveURL(/\/people\/groups\?voice=\d+$/);
    // The notice stays on screen in the detail, and nothing is called validated.
    await expect(page.getByText("no group is ever confirmed")).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Proposed grouping of 2 voices" }),
    ).toBeVisible();
    await expect(page.getByText("2 of 2 voices kept")).toBeVisible();
    await expect(page.getByText(/validated/i)).toHaveCount(0);
    await page.getByRole("radio", { name: "Someone new" }).check();
    await page.getByLabel("Name of the new person").fill("Ne pas écrire");

    // Someone else leaves a voice out while the detail is open.
    await changeGroup(page, "PUT", "SPEAKER_01");
    await page
      .getByRole("button", { name: "Name 2 voice(s) as Ne pas écrire" })
      .click();
    await expect(page.getByRole("alert")).toContainText(
      "The group changed while you were looking at it",
    );

    // Nothing was written: no person, no confirmed voice, both voices still unknown.
    const written = await page.evaluate(async () => {
      const persons = await fetch("/api/persons").then((r) => r.json());
      const verification = await fetch("/api/voice-verification").then((r) =>
        r.json(),
      );
      return {
        persons: persons.items.length,
        human: verification.counts.human,
        unknown: verification.counts.unknown,
      };
    });
    expect(written).toEqual({ persons: 0, human: 0, unknown: 2 });

    // Reloading shows the real state: the voice kept is now alone.
    await page.getByRole("button", { name: "Reload" }).click();
    await expect(
      page.getByRole("heading", { name: "Isolated voice" }),
    ).toBeVisible();
    await expect(page.getByText("1 of 1 voices kept")).toBeVisible();
    await expect(page.getByRole("alert")).toHaveCount(0);

    await page.getByRole("button", { name: "Back to groups" }).click();
    await expect(page.getByText("No group left to name.")).toBeVisible();
    await changeGroup(page, "DELETE", "SPEAKER_01");
  });

  test("groupes en thème sombre et à 390 pixels", async () => {
    await page.evaluate(() => localStorage.setItem("snitch-ui-theme", "dark"));
    await page.setViewportSize({ width: 390, height: 780 });
    await page.goto("/people/groups");
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
    await expectNoViolations(page);
    await page.getByRole("button", { name: /^Proposed grouping of/ }).click();
    await expect(
      page.getByRole("heading", { name: "Proposed grouping of 2 voices" }),
    ).toBeVisible();
    await expectNoViolations(page);
    // The bar of action stays within reach and each control is 44 px tall.
    await expect(page.getByRole("region", { name: "Naming" })).toBeInViewport();
    const heights = await page
      .getByRole("checkbox")
      .evaluateAll((all) =>
        all.map((e) => e.closest("label")!.getBoundingClientRect().height),
      );
    expect(heights.length).toBe(2);
    for (const height of heights) expect(height).toBeGreaterThanOrEqual(44);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
  });

  test("écarter, remettre puis nommer valide les voix retenues", async () => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.evaluate(() => localStorage.removeItem("snitch-ui-theme"));
    await page.goto("/people/groups");
    const card = page
      .getByRole("list", { name: "Groups of unknown voices" })
      .locator("xpath=./li");
    await card.getByRole("button", { name: /^Proposed grouping of/ }).click();
    const voice = new URL(page.url()).searchParams.get("voice")!;

    // Every excerpt is there to play, served to the session, with its level.
    const voices = page.getByRole("list", { name: "Voices of the group" });
    await expect(voices.locator("xpath=./li")).toHaveCount(2);
    await expect(
      page.getByRole("button", {
        name: "Excerpt of SPEAKER_00 in synthetic-voices.mp4",
      }),
    ).toBeVisible();
    await expect(voices.locator("xpath=./li").nth(0)).toContainText("Unknown");
    const source = await page.locator("audio").first().getAttribute("src");
    expect(source).toMatch(/^\/api\/local-speakers\/\d+\/excerpt$/);
    expect((await page.request.get(source!)).headers()["content-type"]).toBe(
      "audio/wav",
    );

    // The button names nothing until a person is chosen.
    await expect(
      page.getByRole("button", { name: "Name these voices" }),
    ).toBeDisabled();

    // Leaving a voice out changes only that voice; putting it back restores the count.
    const second = page.getByRole("checkbox", { name: /SPEAKER_01/ });
    await second.click();
    await expect(second).not.toBeChecked();
    await expect(page.getByText("1 of 2 voices kept")).toBeVisible();
    await expect(voices.locator("xpath=./li").nth(1)).toContainText("Left out");
    await expect(voices.locator("xpath=./li").nth(0)).not.toContainText(
      "Left out",
    );
    await expect(
      page.getByRole("checkbox", { name: /SPEAKER_00/ }),
    ).toBeChecked();
    await second.click();
    await expect(second).toBeChecked();
    await expect(page.getByText("2 of 2 voices kept")).toBeVisible();

    // One voice left out, the other named in one action.
    await second.click();
    await expect(second).not.toBeChecked();
    await expect(page.getByText("1 of 2 voices kept")).toBeVisible();
    await page.getByRole("radio", { name: "Someone new" }).check();
    await page.getByLabel("Name of the new person").fill("Groupe Synthétique");
    await expectNoViolations(page);
    await page
      .getByRole("button", { name: "Name 1 voice(s) as Groupe Synthétique" })
      .click();
    await expect(page.getByRole("status").last()).toContainText(
      "Saved: 1 voice(s) are now Groupe Synthétique. 1 left out.",
    );

    // Only the kept voice was validated; the left-out one is at most a suggestion.
    const counts = await page.evaluate(async () => {
      const verification = await fetch("/api/voice-verification").then((r) =>
        r.json(),
      );
      return {
        human: verification.counts.human,
        suggestion: verification.counts.suggestion,
      };
    });
    expect(counts.human).toBe(1);
    expect(counts.suggestion).toBe(1);
    await expect(page.getByText("No group left to name.")).toBeVisible();

    // The address of the voice that was named says it was handled; nothing else opens.
    const gone = page.getByText("Already handled or no longer visible");
    await page.goto(`/people/groups?voice=${voice}`);
    await expect(gone).toBeVisible();
    await expect(
      page.getByRole("list", { name: "Voices of the group" }),
    ).toHaveCount(0);
    await page.goto("/people/groups?voice=99999999");
    await expect(gone).toBeVisible();
  });
});
