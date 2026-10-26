import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";

// The guided installation, from the installation code to the summary, then the banner that
// resumes it. The instance is brand new: nothing is prepared through the API, every step
// is done in the browser. The Hugging Face token must never reach Snitch.

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];
// Built at run time so that no secret scanner mistakes this synthetic value for a real token.
const HF_TOKEN = ["hf", "synthetic0123456789abcdefghijklmnop"].join("_");
const OWNER = {
  username: "first-owner",
  password: "synthetic-password-123",
};

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

test.describe.serial("assistant d'installation", () => {
  test("code, compte, étapes facultatives et fermeture de l'onglet", async ({
    browser,
  }) => {
    const context = await browser.newContext();
    const leaked: string[] = [];
    context.on("request", (request) => {
      if (!request.url().includes("/api/")) return;
      const sent = `${request.url()}\n${request.postData() ?? ""}`;
      if (sent.includes(HF_TOKEN)) leaked.push(request.url());
    });
    let page = await context.newPage();

    // The first screen says where to read the code and never shows it.
    await page.goto("/");
    await expect(
      page.getByRole("heading", { name: "Enter the installation code" }),
    ).toBeVisible();
    await expect(page.getByText("docker compose logs app")).toBeVisible();
    await expect(page.locator("body")).not.toContainText(
      process.env.SNITCH_TEST_CODE!,
    );
    await expectNoViolations(page);

    // On a phone: no sideways scroll, and still accessible.
    await page.setViewportSize({ width: 390, height: 780 });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    await expectNoViolations(page);
    await page.setViewportSize({ width: 1280, height: 800 });

    // The wizard follows the language choice.
    await page.getByLabel("Language").selectOption("fr");
    await expect(
      page.getByRole("heading", { name: "Saisissez le code d’installation" }),
    ).toBeVisible();
    await page.getByLabel("Langue").selectOption("en");

    // A wrong code is flagged at once, before any password is asked for.
    await page.getByLabel("Installation code").fill("not-the-code");
    await page.getByRole("button", { name: "Continue" }).click();
    await expect(page.getByRole("alert")).toContainText("wrong or has expired");
    await expect(
      page.getByRole("heading", { name: "Create the Super-administrator" }),
    ).toHaveCount(0);
    await expectNoViolations(page);

    await page
      .getByLabel("Installation code")
      .fill(process.env.SNITCH_TEST_CODE!);
    await page.getByRole("button", { name: "Continue" }).click();
    await expect(
      page.getByRole("heading", { name: "Create the Super-administrator" }),
    ).toBeVisible();

    // Going back and forth loses nothing; the two passwords must match.
    await page.getByLabel("Username").fill(OWNER.username);
    await page.getByLabel("Password", { exact: true }).fill(OWNER.password);
    await page.getByLabel("Confirm the password").fill("another-password-9");
    await expect(
      page.getByText("The two passwords do not match."),
    ).toBeVisible();
    await page.getByRole("button", { name: "Back" }).click();
    await expect(page.getByLabel("Installation code")).toHaveValue(
      process.env.SNITCH_TEST_CODE!,
    );
    await page.getByRole("button", { name: "Continue" }).click();
    await expect(page.getByLabel("Username")).toHaveValue(OWNER.username);
    await page.getByLabel("Confirm the password").fill(OWNER.password);
    await expectNoViolations(page);
    await page.getByRole("button", { name: "Create the account" }).click();

    // Discord: configured for real (Snitch does not call Discord to save it).
    await expect(
      page.getByRole("heading", { name: "Sign in with Discord" }),
    ).toBeVisible();
    await expect(page).toHaveURL(/\/setup$/);
    const steps = page.getByRole("list", { name: "Setup steps" });
    await expect(steps.getByRole("listitem")).toHaveCount(7);
    await expect(steps.getByRole("listitem").nth(2)).toHaveAttribute(
      "aria-current",
      "step",
    );
    await expect(
      page.getByLabel("Return address to register in Discord"),
    ).toHaveValue(`${process.env.SNITCH_TEST_URL}/api/auth/discord/callback`);
    await expectNoViolations(page);
    await page.getByLabel("Application ID").fill("123456789012345678");
    await page.getByLabel("Application secret").fill("synthetic-secret");
    await page.getByRole("button", { name: "Save and continue" }).click();
    await expect(
      page.getByRole("heading", { name: "Semantic search" }),
    ).toBeVisible();
    await expectNoViolations(page);

    // The tab is closed here: the next steps are still to do.
    await page.close();
    page = await context.newPage();
    await page.goto("/library");
    const banner = page.getByRole("region", { name: "Setup" });
    await expect(banner).toContainText("Finish the setup");
    await expectNoViolations(page);
    await banner.getByRole("link", { name: "Resume the setup" }).click();
    await expect(
      page.getByRole("heading", { name: "Semantic search" }),
    ).toBeVisible();
    await expect(
      page
        .getByRole("list", { name: "Setup steps" })
        .getByRole("listitem")
        .nth(2),
    ).toContainText("Done");

    // Skipping is always possible.
    await page.getByRole("button", { name: "Skip for now" }).click();
    await expect(
      page.getByRole("heading", { name: "Create your first Worker" }),
    ).toBeVisible();
    await expectNoViolations(page);

    // The first Worker: the same form as the administration, with a Skip button next to it.
    await expect(
      page.getByRole("button", { name: "Skip for now" }),
    ).toBeVisible();
    await page.getByLabel("Worker name").fill("gaming-pc");
    await page
      .getByLabel("Whisper server address")
      .fill("http://host.docker.internal:8000");
    await page.getByLabel("Whisper model name").fill("whisper-large-v3");
    await page.getByRole("main").getByLabel("Language").selectOption("fr");
    await page.getByLabel("Hugging Face token").fill(HF_TOKEN);
    await page.getByRole("button", { name: "Create the Worker" }).click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toContainText("whisper-large-v3 · fr");
    await page.getByRole("button", { name: "Confirm and create" }).click();
    await expect(page.getByText("Copy both files now.")).toBeVisible();
    const env = await page
      .getByRole("textbox", { name: "Content of .env" })
      .inputValue();
    expect(env).toContain("HF_TOKEN=" + HF_TOKEN);
    expect(env).toMatch(/^SNITCH_WORKER_TOKEN=rw_\S+$/m);
    await expectNoViolations(page);
    await page.getByRole("button", { name: "I copied the files" }).click();

    await expect(
      page.getByRole("heading", { name: "Import your first videos" }),
    ).toBeVisible();
    await expect(
      page.getByText("Nothing is scanned or processed"),
    ).toBeVisible();
    await expectNoViolations(page);
    await page.getByRole("button", { name: "Skip for now" }).click();

    // The summary tells what is done and where to finish what was skipped.
    await expect(page.getByRole("heading", { name: "Summary" })).toBeVisible();
    const row = (name: string) => page.getByRole("row", { name });
    await expect(row("Super-administrator")).toContainText("Done");
    await expect(row("Discord")).toContainText("Done");
    await expect(row("First Worker")).toContainText("Done");
    await expect(row("Semantic search")).toContainText("Skipped");
    await expect(row("Semantic search")).toContainText(
      "Finish it in Administration, Semantic index",
    );
    await expect(row("Import videos")).toContainText("Skipped");
    await expectNoViolations(page);

    // Nothing started by itself: no task was queued and the library is empty.
    const jobs = await (await page.request.get("/api/admin/jobs")).json();
    expect(jobs.items ?? []).toHaveLength(0);
    const clips = await (await page.request.get("/api/clips")).json();
    expect(clips.items ?? []).toHaveLength(0);

    await page.getByRole("button", { name: "Open the library" }).click();
    await expect(page).toHaveURL(/\/library$/);
    await expect(page.getByRole("region", { name: "Setup" })).toContainText(
      "Finish the setup",
    );
    expect(leaked).toEqual([]);

    // The account of the wizard is the Super-administrator.
    await page.getByRole("button", { name: "Account menu" }).click();
    await expect(page.getByRole("menu")).toContainText("Super-administrator");
    await page.keyboard.press("Escape");

    // The banner can be closed for good: it stays closed after a reload and on a new session.
    await page
      .getByRole("button", { name: "Hide this banner for good" })
      .click();
    await expect(page.getByRole("region", { name: "Setup" })).toHaveCount(0);
    await page.reload();
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
    await expect(page.getByRole("region", { name: "Setup" })).toHaveCount(0);
    await context.close();

    const again = await browser.newContext();
    const next = await again.newPage();
    await next.goto("/login");
    await next.getByLabel("Username").fill(OWNER.username);
    await next.getByLabel("Password").fill(OWNER.password);
    await next.getByRole("button", { name: "Sign in" }).click();
    await expect(next).toHaveURL(/\/library$/);
    await expect(next.getByRole("heading", { level: 1 })).toBeVisible();
    await expect(next.getByRole("region", { name: "Setup" })).toHaveCount(0);
    await again.close();
  });

  test("une fois installé, le code ne sert plus et l'assistant reste réservé aux administrateurs", async ({
    page,
  }) => {
    await page.goto("/setup");
    await expect(page).toHaveURL(/\/login$/);
    const refused = await page.request.post("/api/setup/code", {
      data: { code: process.env.SNITCH_TEST_CODE! },
      headers: { Origin: process.env.SNITCH_TEST_URL! },
    });
    expect(refused.status()).toBe(409);
  });
});
