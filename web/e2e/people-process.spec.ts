import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import {
  adminApi,
  nameVoice,
  transcribeAndDiarize,
  uploadClip,
} from "./library-fixture";
import { prepareDiarizedClip } from "./voice-fixture";

// The processing screen of one clip (Lot C): the waveform, the queue of voices, the card
// "Who is speaking here?", the keyboard shortcuts and the way to the next clip. Synthetic
// clips of the shared fixture only; everything goes through the screen or the API.

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

type Voice = {
  id: number;
  label: string;
  disposition?: string | null;
  person: { display_name: string } | null;
};
async function voicesOf(clipId: number) {
  const { api } = await adminApi();
  const voice = (await (
    await api.get(`/api/clips/${clipId}/voice`)
  ).json()) as {
    speakers: Voice[];
  };
  await api.dispose();
  return voice.speakers;
}

let page: Page;
let clipId: number;

test.describe.serial("personnes : écran de traitement d'un clip", () => {
  test.beforeAll(async ({ browser }) => {
    page = await (await browser.newContext()).newPage();
    clipId = await prepareDiarizedClip(page, {
      secondFingerprintEnd: 4.2,
      peaks: true,
    });
    // The synthetic excerpt cannot be decoded: the media element is stubbed and what is
    // observed is which source was asked for.
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
  });
  test.afterAll(async () => {
    await page.context().close();
  });

  test("Traiter ouvre l'écran du clip : forme d'onde, file, extraits, raccourcis et voix suivante", async () => {
    await page.goto("/people/verification");
    await page.getByRole("link", { name: /^Process / }).click();
    await expect(page).toHaveURL(
      new RegExp(`/people/verification\\?clip=${clipId}$`),
    );
    const screen = page.getByRole("region", { name: "Clip processing" });
    await expect(screen).toBeVisible();
    await expect(
      screen.getByRole("list", { name: "Tracks of the clip, one per voice" }),
    ).toBeVisible();
    const queue = screen.getByRole("list", { name: "Voices to process" });
    await expect(queue.getByRole("listitem")).toHaveCount(2);
    await expect(
      screen.getByRole("link", { name: "Open the clip sheet" }),
    ).toHaveAttribute("href", `/clips/${clipId}`);

    const card = screen.getByRole("region", { name: "Who is speaking here?" });
    await expect(card.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_00",
    );
    // The excerpts carry their text and a way to listen.
    const excerpts = card.getByRole("list", { name: "Excerpts of the voice" });
    await expect(excerpts).toContainText("Bien joué !");
    await expectNoViolations(page);

    // Space listens to the excerpt of the voice.
    await page.keyboard.press("Space");
    const played = () =>
      page.evaluate(() => (window as unknown as { played: string[] }).played);
    expect(await played()).toEqual([
      expect.stringMatching(/^\/api\/local-speakers\/\d+\/excerpt$/),
    ]);

    // The shortcuts are off in a text field: the letters are typed, nothing is decided.
    await card.getByRole("button", { name: "Identify" }).click();
    const search = card.getByRole("searchbox", { name: "Search people" });
    await search.fill("");
    await search.press("r");
    await search.press("i");
    await search.press("n");
    await expect(search).toHaveValue("rin");
    await expect(card.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_00",
    );
    expect((await voicesOf(clipId)).every((v) => !v.disposition)).toBe(true);

    // A new Person settles the voice, and the next one comes by itself.
    await card.getByLabel("Name of the new person").fill("Claire Synthétique");
    await card.getByRole("button", { name: "Create and identify" }).click();
    await expect(card.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_01",
    );
    // The second voice now looks like the first: a suggestion with its proximity.
    await expect(card).toContainText("% proximity");
    await expect(
      card.getByRole("button", { name: "It is Claire Synthétique" }),
    ).toBeVisible();

    // N refuses the suggestion and keeps the voice to decide.
    await page.keyboard.press("n");
    await expect(
      card.getByRole("button", { name: "It is Claire Synthétique" }),
    ).toHaveCount(0);
    await expect(card.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_01",
    );

    // R sets it to Random: the clip is done.
    await page.keyboard.press("r");
    await expect(
      screen.getByText("All the voices of this clip are processed."),
    ).toBeVisible();
    const voices = await voicesOf(clipId);
    expect(voices.find((v) => v.label === "SPEAKER_01")!.disposition).toBe(
      "random",
    );
    await expect(screen.getByText("No other clip to process.")).toBeVisible();

    // Handled, the clip leaves the list.
    await page.goto("/people/verification");
    await expect(
      page
        .getByRole("region", { name: "Clips to process" })
        .getByRole("listitem"),
    ).toHaveCount(0);
  });

  test("candidats, Passer, Fausse détection, ?case= et Vidéo suivante", async () => {
    const second = await uploadClip("second-voices.mp4", "shared");
    await transcribeAndDiarize(second, { secondFingerprintEnd: 4.2 });
    const fourth = await uploadClip("fourth-voices.mp4", "shared");
    await transcribeAndDiarize(fourth, { secondFingerprintEnd: 4.2 });
    // Two references close to every voice of the new clips: those voices are ambiguous and
    // the server proposes both Persons as candidates.
    const third = await uploadClip("third-voices.mp4", "shared");
    const thirdVoices = await transcribeAndDiarize(third, {
      secondFingerprintEnd: 4.2,
    });
    await nameVoice(thirdVoices[0], "Référence Synthétique A");
    await nameVoice(thirdVoices[1], "Référence Synthétique B");

    // The old address of a case opens the screen of the clip of that case, on that voice.
    const target = (await voicesOf(second)).find(
      (v) => v.label === "SPEAKER_01",
    )!;
    await page.goto(`/people/verification?case=${target.id}`);
    const screen = page.getByRole("region", { name: "Clip processing" });
    const card = screen.getByRole("region", { name: "Who is speaking here?" });
    await expect(card.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_01",
    );
    await expect(screen).toContainText("second-voices.mp4");
    await expect(
      screen
        .getByRole("list", { name: "Voices to process" })
        .getByRole("listitem"),
    ).toHaveCount(2);

    // The closest people come with their proximity, never as a confirmation.
    await expect(card).toContainText("% proximity");
    await expect(
      card
        .getByRole("button", { name: /^It is Référence Synthétique/ })
        .first(),
    ).toBeVisible();
    await expectNoViolations(page);

    // Skip only moves on in this session: nothing is written.
    await card.getByRole("button", { name: "Skip" }).click();
    await expect(card.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_00",
    );
    expect(
      (await voicesOf(second)).every((v) => !v.disposition && !v.person),
    ).toBe(true);

    // I is a false detection: the voice is ignored, the next one comes.
    await page.keyboard.press("i");
    await expect(card.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_01",
    );
    expect(
      (await voicesOf(second)).find((v) => v.label === "SPEAKER_00")!
        .disposition,
    ).toBe("ignored");
    await page.keyboard.press("r");
    await expect(
      screen.getByText("All the voices of this clip are processed."),
    ).toBeVisible();

    // Next video goes to another clip that still has a voice to process.
    await screen.getByRole("button", { name: "Next video" }).click();
    await expect(page).toHaveURL(
      new RegExp(`/people/verification\\?clip=${fourth}$`),
    );
    await expect(
      screen.getByRole("region", { name: "Who is speaking here?" }),
    ).toBeVisible();

    // The notification address stays valid, and a clip without voice to process says so.
    await page.goto(`/people/verification?clip=${clipId}`);
    await expect(
      screen.getByText("All the voices of this clip are processed."),
    ).toBeVisible();
  });

  test("à 390 pixels en thème sombre", async () => {
    await page.evaluate(() => localStorage.setItem("snitch-ui-theme", "dark"));
    await page.setViewportSize({ width: 390, height: 780 });
    await page.goto("/people/verification");
    await page.getByRole("link", { name: /^Process / }).click();
    const card = page.getByRole("region", { name: "Who is speaking here?" });
    await expect(card).toBeVisible();
    await expectNoViolations(page);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    // The decisions are comfortable to hit (the state badge is a legend, not a decision).
    const decisions = card
      .getByRole("button")
      .filter({ hasNotText: /^(Unknown|Ambiguous|Suggestion)$/ });
    for (const button of await decisions.all()) {
      expect((await button.boundingBox())!.height).toBeGreaterThanOrEqual(40);
    }
  });
});
