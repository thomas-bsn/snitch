import { expect, test, type Page } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// Unknown voices: Random and Ignore for the whole group, from the naming panel (ticket 222).

let page: Page;

test.describe.serial("groupe de voix : disposition", () => {
  test.beforeAll(async ({ browser }) => {
    page = await (await browser.newContext()).newPage();
    await prepareDiarizedClip(page, { secondFingerprintEnd: 4.2 });
  });
  test.afterAll(async () => {
    await page.context().close();
  });

  test("un groupe modifié entre-temps est refusé sans rien écrire", async () => {
    await page.goto("/people/groups");
    await page.getByRole("button", { name: /^Listen and name/ }).click();
    await expect(
      page.getByRole("heading", { name: "Proposed grouping of 2 voices" }),
    ).toBeVisible();

    // Someone else leaves a voice out while the panel is open.
    const exclusion = (verb: "PUT" | "DELETE") =>
      page.evaluate(async (method) => {
        const me = await fetch("/api/auth/me").then((r) => r.json());
        const groups = await fetch(
          "/api/voice-groups?proposed=false&page_size=100",
        ).then((r) => r.json());
        const id = groups.items
          .flatMap(
            (g: { members: { label: string; local_speaker_id: number }[] }) =>
              g.members,
          )
          .find(
            (m: { label: string }) => m.label === "SPEAKER_01",
          ).local_speaker_id;
        await fetch(`/api/local-speakers/${id}/group-exclusion`, {
          method,
          headers: { "X-CSRF-Token": me.csrf_token, Origin: location.origin },
        });
      }, verb);
    await exclusion("PUT");

    await page.getByRole("button", { name: "Ignore the whole group" }).click();
    await expect(page.getByRole("alert")).toContainText(
      "The group changed while you were looking at it",
    );
    const dispositions = await page.evaluate(async () => {
      const v = await fetch("/api/voice-verification").then((r) => r.json());
      return v.counts.unknown;
    });
    expect(dispositions).toBe(2);

    await exclusion("DELETE");
  });

  test("ignorer tout le groupe le retire de la liste", async () => {
    await page.goto("/people/groups");
    await page.getByRole("button", { name: /^Listen and name/ }).click();
    await expect(
      page.getByRole("heading", { name: "Proposed grouping of 2 voices" }),
    ).toBeVisible();
    await expect(page.getByText("Heard in 1 video(s)")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Random for the whole group" }),
    ).toBeVisible();

    await page.getByRole("button", { name: "Ignore the whole group" }).click();
    await expect(page.getByRole("status").last()).toContainText(
      "Ignored: 2 voice(s) set aside.",
    );
    await expect(page.getByText("No group left to name.")).toBeVisible();
    const unknown = await page.evaluate(async () => {
      const v = await fetch("/api/voice-verification").then((r) => r.json());
      return v.counts.unknown;
    });
    expect(unknown).toBe(0);
  });
});
