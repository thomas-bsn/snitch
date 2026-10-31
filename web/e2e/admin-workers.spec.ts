import AxeBuilder from "@axe-core/playwright";
import { expect, request, test, type Page } from "@playwright/test";
import {
  ADMIN,
  MEMBER,
  createApprovedMember,
  installAdministrator,
  sessionFor,
} from "./shell-fixture";

// Administration, Workers and the page of a Worker, in the new interface. The Hugging Face
// token must never reach Snitch: every request of the journey is inspected.

// Built at run time so that no secret scanner mistakes this synthetic value for a real token.
const HF_TOKEN = ["hf", "synthetic0123456789abcdefghijklmnop"].join("_");
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

/** A Worker pulse, as the real Worker would send it with its token. */
async function pulse(token: string) {
  const api = await request.newContext({
    baseURL: process.env.SNITCH_TEST_URL,
  });
  const response = await api.post("/api/worker/pulse", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      capacity: 1,
      supported_steps: ["fixture.*"],
      machine: {
        name: "Synthetic host",
        system: "Linux",
        processor: "Synthetic CPU",
        memory_bytes: 8589934592,
        gpu_name: "Synthetic GPU",
        gpu_memory_bytes: 4294967296,
        gpu_driver: "555.1",
        worker_version: "1.2.3",
      },
      models: {
        transcription: {
          provider: "openai-compatible",
          model: "whisper-large-v3",
        },
        diarization: {
          provider: "pyannote",
          model: "speaker-diarization-community-1",
        },
      },
    },
  });
  const status = response.status();
  await api.dispose();
  return status;
}

test.describe.serial("administration et Workers", () => {
  test.beforeAll(async () => {
    await installAdministrator();
    await createApprovedMember();
  });

  let firstToken = "";
  let firstId = "";
  const leaked: string[] = [];

  test("un membre ordinaire ne voit pas l'administration", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await context.newPage();
    await page.goto("/library");
    await page.getByRole("button", { name: "Account menu" }).click();
    await expect(page.getByRole("menu")).toBeVisible();
    await expect(
      page.getByRole("menuitem", { name: "Administration" }),
    ).toHaveCount(0);
    await page.keyboard.press("Escape");
    await page.goto("/admin/workers");
    await expect(
      page.getByRole("heading", { name: "Administrators only" }),
    ).toBeVisible();
    await expect(
      page.getByRole("navigation", { name: "Administration sections" }),
    ).toHaveCount(0);
    await context.close();
  });

  test("configurer la transcription avant le premier Worker", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/admin/transcription");
    // Initially unconfigured
    await expect(
      page.getByText("Transcription needs configuration."),
    ).toBeVisible();
    // Configure transcription
    await page
      .getByLabel("Whisper/Speaches server URL")
      .fill("http://host.docker.internal:8000");
    await page.getByLabel("Library model").fill("whisper-large-v3");
    await page.getByLabel("Language").fill("fr");
    await page.getByRole("button", { name: "Save and version" }).click();
    await expect(
      page.getByText("Saved, not yet tested from a Worker."),
    ).toBeVisible();
    await expectNoViolations(page);
    await context.close();
  });

  test("la barre d'onglets, le premier Worker et son jeton Hugging Face", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
      permissions: ["clipboard-read", "clipboard-write"],
    });
    const page = await context.newPage();
    page.on("request", (request) => {
      if (!request.url().includes("/api/")) return;
      const sent = `${request.url()}\n${request.postData() ?? ""}\n${JSON.stringify(request.headers())}`;
      if (sent.includes(HF_TOKEN)) leaked.push(request.url());
    });
    await page.goto("/library");
    await page.getByRole("button", { name: "Account menu" }).click();
    await page.getByRole("menuitem", { name: "Administration" }).click();
    await expect(page).toHaveURL(/\/admin\/workers$/);

    const tabs = page.getByRole("navigation", {
      name: "Administration sections",
    });
    await expect(tabs.getByRole("link")).toHaveText([
      "Workers",
      "Task queue",
      "Folders",
      "Location and metadata",
      "Transcription",
      "Voice processing",
      "Semantic index",
      "Users and access",
      "Exports",
    ]);
    await expect(tabs.getByRole("link", { name: "Workers" })).toHaveAttribute(
      "aria-current",
      "page",
    );
    await expect(page.getByText("No Worker yet.")).toBeVisible();
    await expectNoViolations(page);

    await page.getByRole("link", { name: "Add a Worker" }).click();
    await expect(page).toHaveURL(/\/admin\/workers\/new$/);
    // The address of Snitch is filled in and read-only; plain HTTP is flagged.
    const snitch = page.getByLabel("Snitch address");
    await expect(snitch).toHaveValue(process.env.SNITCH_TEST_URL!);
    await expect(snitch).toHaveAttribute("readonly", "");
    await expect(page.getByText("This address is not HTTPS")).toBeVisible();
    // Transcription settings are read-only and pre-filled from central config.
    const whisperEndpoint = page.getByLabel("Whisper server address");
    await expect(whisperEndpoint).toHaveValue(
      "http://host.docker.internal:8000",
    );
    await expect(whisperEndpoint).toHaveAttribute("readonly", "");
    const libraryModel = page.getByLabel("Library model");
    await expect(libraryModel).toHaveValue("whisper-large-v3");
    await expect(libraryModel).toHaveAttribute("readonly", "");
    const language = page.getByLabel("Language");
    await expect(language).toHaveValue("fr");
    await expect(language).toHaveAttribute("readonly", "");
    await expectNoViolations(page);

    await page.getByLabel("Worker name").fill("gaming-pc");
    await page.getByLabel("Hugging Face token").fill(HF_TOKEN);
    await page.getByRole("button", { name: "Create the Worker" }).click();

    // The first Worker engages the whole library: a confirmation says so.
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toContainText("every future Worker");
    await expect(dialog).toContainText("whisper-large-v3 · fr");
    await expectNoViolations(page);
    await page.keyboard.press("Escape");
    await expect(dialog).toHaveCount(0);
    await page.getByRole("button", { name: "Create the Worker" }).click();
    await page.getByRole("button", { name: "Confirm and create" }).click();

    // Files: the token of the Worker is shown once; the Hugging Face token is in the .env only.
    await expect(page.getByText("Copy both files now.")).toBeVisible();
    const env = page.getByRole("textbox", { name: "Content of .env" });
    const text = await env.inputValue();
    expect(text).toContain("HF_TOKEN=" + HF_TOKEN);
    expect(text).toContain(
      "TRANSCRIPTION_API_URL=http://host.docker.internal:8000",
    );
    expect(text).toContain("TRANSCRIPTION_MODEL=whisper-large-v3");
    expect(text).toContain("TRANSCRIPTION_LANGUAGE=fr");
    firstToken = /^SNITCH_WORKER_TOKEN=(\S+)$/m.exec(text)![1];
    expect(firstToken).toMatch(/^rw_/);
    await expectNoViolations(page);

    await page.getByRole("button", { name: "Copy" }).first().click();
    await expect(page.getByRole("button", { name: "Copied" })).toBeVisible();
    expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
      text,
    );

    const download = page.waitForEvent("download");
    await page
      .getByRole("button", { name: /Download snitch-worker\.env/ })
      .click();
    expect((await download).suggestedFilename()).toBe("snitch-worker.env");

    await page.getByRole("tab", { name: "compose.yaml" }).click();
    const compose = await page
      .getByRole("textbox", { name: "Content of compose.yaml" })
      .inputValue();
    expect(compose).toContain("TRANSCRIPTION_MODEL: ${TRANSCRIPTION_MODEL:-}");
    expect(compose).not.toContain(firstToken);

    await page.getByRole("button", { name: "I copied the files" }).click();
    await expect(page).toHaveURL(/\/admin\/workers\/wk_/);
    firstId = page.url().split("/").pop()!;
    expect(leaked).toEqual([]);
    await context.close();
  });

  test("le Worker suivant affiche les réglages de transcription en lecture seule", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/admin/workers/new");
    const whisperEndpoint = page.getByLabel("Whisper server address");
    await expect(whisperEndpoint).toHaveValue(
      "http://host.docker.internal:8000",
    );
    await expect(whisperEndpoint).toHaveAttribute("readonly", "");
    const libraryModel = page.getByLabel("Library model");
    await expect(libraryModel).toHaveValue("whisper-large-v3");
    await expect(libraryModel).toHaveAttribute("readonly", "");
    const language = page.getByLabel("Language");
    await expect(language).toHaveValue("fr");
    await expect(language).toHaveAttribute("readonly", "");
    await expectNoViolations(page);
    await context.close();
  });

  test("la fiche du Worker, le nouveau jeton et la révocation", async ({
    browser,
  }) => {
    expect(await pulse(firstToken)).toBe(200);
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/admin/workers");
    await expect(page.getByRole("row", { name: /gaming-pc/ })).toContainText(
      "Online",
    );
    await expect(page.getByRole("row", { name: /gaming-pc/ })).toContainText(
      "whisper-large-v3",
    );
    await page.getByRole("link", { name: "Open gaming-pc" }).click();
    await expect(page).toHaveURL(new RegExp(`/admin/workers/${firstId}$`));

    await expect(
      page.getByRole("heading", { name: "gaming-pc" }),
    ).toBeVisible();
    await expect(page.getByText("Online")).toBeVisible();
    await expect(page.getByText("1 at a time")).toBeVisible();
    await expect(
      page.getByText("This Worker has not done any task yet."),
    ).toBeVisible();
    const models = page.getByRole("region", { name: "Models in use" });
    await expect(models).toContainText("whisper-large-v3");
    await expect(models).toContainText("speaker-diarization-community-1");
    await expect(models).toContainText("Not declared");
    const machine = page.getByRole("region", { name: "Machine" });
    await expect(machine).toContainText("Synthetic host");
    await expect(machine).toContainText("Synthetic CPU");
    await expect(machine).toContainText("Synthetic GPU");
    await expect(machine).toContainText("555.1");
    await expect(machine).toContainText("1.2.3");
    // The token is never shown again: the file holds a placeholder.
    const env = page.getByRole("textbox", { name: "Content of .env" });
    expect(await env.inputValue()).not.toContain(firstToken);
    expect(await env.inputValue()).toContain(
      "SNITCH_WORKER_TOKEN=A_RENSEIGNER",
    );
    await expectNoViolations(page);

    // A new token: confirmation, then the old one stops working.
    await page.getByRole("button", { name: "Generate a new token…" }).click();
    await expect(page.getByRole("alertdialog")).toContainText(
      "stops working immediately",
    );
    await page.getByRole("button", { name: "Cancel" }).click();
    expect(await pulse(firstToken)).toBe(200);
    await page.getByRole("button", { name: "Generate a new token…" }).click();
    await page.getByRole("button", { name: "Generate", exact: true }).click();
    await expect(page.getByText("New token generated.")).toBeVisible();
    const renewed = /^SNITCH_WORKER_TOKEN=(\S+)$/m.exec(
      await env.inputValue(),
    )![1];
    expect(renewed).toMatch(/^rw_/);
    expect(renewed).not.toBe(firstToken);
    expect(await pulse(firstToken)).toBe(401);
    expect(await pulse(renewed)).toBe(200);
    await page.reload();
    expect(
      await page.getByRole("textbox", { name: "Content of .env" }).inputValue(),
    ).not.toContain(renewed);

    // Revocation: confirmation, then the Worker is revoked and its token refused.
    await page.getByRole("button", { name: "Revoke this Worker…" }).click();
    await expect(page.getByRole("alertdialog")).toContainText(
      "Revoke gaming-pc?",
    );
    await expectNoViolations(page);
    await page.getByRole("button", { name: "Revoke", exact: true }).click();
    await expect(page.getByText(/This Worker was revoked on/)).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Revoke this Worker…" }),
    ).toHaveCount(0);
    expect(await pulse(renewed)).toBe(401);
    await context.close();
  });
});
