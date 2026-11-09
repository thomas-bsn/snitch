import AxeBuilder from "@axe-core/playwright";
import {
  expect,
  request,
  test,
  type Browser,
  type Page,
} from "@playwright/test";
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
    const language = page.getByRole("combobox", {
      name: "Language",
      exact: true,
    });
    await language.fill("french");
    await expect(
      page.getByRole("option", { name: "French (fr)" }),
    ).toHaveAttribute("data-highlighted", "");
    await language.press("Enter");
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
      "Updates",
      "Folders",
      "Location and metadata",
      "Transcription",
      "Voice processing",
      "Benchmark",
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
    await expect(language).toHaveValue("French (fr)");
    await expect(language).toHaveAttribute("readonly", "");
    await expectNoViolations(page);

    // Plain HTTP needs an explicit, visibly warned opt-in; without it no command is generated.
    const optIn = page.getByRole("checkbox", {
      name: /Allow an installation command over plain HTTP/,
    });
    await expect(optIn).not.toBeChecked();
    await optIn.check();
    await expect(page.getByText(/unencrypted/)).toBeVisible();
    await page.getByLabel("Worker name").fill("gaming-pc");
    await page.getByRole("button", { name: "Create the Worker" }).click();

    // One command to run; the files are kept under "Advanced". Secrets never go through the form.
    await expect(
      page.getByText("Copy the installation command now."),
    ).toBeVisible();
    const command = await page
      .getByRole("textbox", { name: "Worker installation command" })
      .inputValue();
    expect(command).not.toContain(HF_TOKEN);
    expect(command).toContain("/api/worker/install/");
    await expectNoViolations(page);
    // The guide says where to run it, what to prepare and what to do when the invitation is gone.
    await expect(
      page.getByRole("heading", { name: "One-command install" }),
    ).toBeVisible();
    await expect(page.getByText("open a WSL2 terminal")).toBeVisible();
    await expect(
      page.getByText("Docker is installed and running"),
    ).toBeVisible();
    await expect(
      page.getByText("single-use and expires after 15 minutes"),
    ).toBeVisible();

    await page.getByRole("button", { name: "Copy command" }).click();
    await expect(page.getByRole("button", { name: "Copied" })).toBeVisible();
    expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
      command,
    );

    await page.getByText("Advanced: deployment files").click();
    const env = page.getByRole("textbox", { name: "Content of .env" });
    const text = await env.inputValue();
    // A new Worker downloads no voice model: the file carries no Hugging Face variable at all.
    expect(text).not.toContain("HF_TOKEN");
    expect(text).not.toContain(HF_TOKEN);
    expect(text).toContain(
      "TRANSCRIPTION_API_URL=http://host.docker.internal:8000",
    );
    // Model and language are managed by Snitch once the Worker is connected: not pinned in the file.
    expect(text).not.toMatch(/^TRANSCRIPTION_MODEL=/m);
    expect(text).not.toMatch(/^TRANSCRIPTION_LANGUAGE=/m);
    firstToken = /^SNITCH_WORKER_TOKEN=(\S+)$/m.exec(text)![1];
    expect(firstToken).toMatch(/^rw_/);
    expect(command).not.toContain(firstToken);

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

    await page.getByRole("button", { name: "Done" }).click();
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
    await expect(language).toHaveValue("French (fr)");
    await expect(language).toHaveAttribute("readonly", "");
    await expectNoViolations(page);
    await context.close();
  });

  test("la progression d'installation : échec, reprise, événement reçu plus tard, voix", async ({
    browser,
  }) => {
    const sendEvents = async (events: object[]) => {
      const api = await request.newContext({
        baseURL: process.env.SNITCH_TEST_URL,
      });
      const response = await api.post("/api/worker/install-events", {
        headers: { Authorization: `Bearer ${firstToken}` },
        data: { events },
      });
      const status = response.status();
      await api.dispose();
      return status;
    };
    const at = (minutesAgo: number) =>
      new Date(Date.now() - minutesAgo * 60_000).toISOString();
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    const open = async () => {
      await page.goto(`/admin/workers/${firstId}`);
      return page.getByRole("region", { name: "Installation" });
    };

    // A failure names the phase, the cause and the recovery action; transcription is shown apart.
    expect(
      await sendEvents([
        {
          phase: "image",
          status: "failed",
          cause_code: "image_pull_failed",
          occurred_at: at(0),
        },
      ]),
    ).toBe(204);
    let section = await open();
    await expect(section).toContainText(
      "Failed at Image: Image download failed",
    );
    await expect(section).toContainText("Recovery action");
    await expect(section).toContainText(
      "Check the network and registry access",
    );
    await expect(section).toContainText("Transcription");
    await expectNoViolations(page);

    // An event queued offline keeps its original time and is marked as received later.
    expect(
      await sendEvents([
        { phase: "prerequisites", status: "succeeded", occurred_at: at(180) },
        { phase: "image", status: "succeeded", occurred_at: at(0) },
      ]),
    ).toBe(204);
    section = await open();
    await expect(section).toContainText("Installation");
    await expect(section).toContainText("received later");
    await expect(section).not.toContainText("Failed at");

    // Voice is unavailable with its cause while the Worker is not even connected yet.
    expect(
      await sendEvents([
        {
          phase: "voice-models",
          status: "failed",
          cause_code: "hf_access_denied",
          occurred_at: at(0),
        },
      ]),
    ).toBe(204);
    section = await open();
    await expect(section).toContainText(
      "Unavailable: Hugging Face access denied",
    );
    await context.close();
  });

  test("le diagnostic d'un Worker jamais connecté tient sur mobile et reste accessible (#263)", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
      viewport: { width: 375, height: 700 },
    });
    const page = await context.newPage();
    const me = await (await page.request.get("/api/auth/me")).json();
    const created = await page.request.post("/api/admin/workers", {
      headers: {
        Origin: process.env.SNITCH_TEST_URL!,
        "X-CSRF-Token": me.csrf_token,
      },
      data: { name: "never-seen", max_concurrency: 1 },
    });
    expect(created.status()).toBe(201);
    const worker = await created.json();
    await page.goto(`/admin/workers/${worker.id}`);
    await page.getByRole("button", { name: "Diagnose / Reconnect" }).click();
    const diagnosis = page.getByRole("region", {
      name: "Connection and recovery",
    });
    await expect(
      diagnosis.getByText(`docker restart snitch-worker-${worker.id}`),
    ).toBeVisible();
    await expect(diagnosis).toContainText("Never");
    await expectNoViolations(page);
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - window.innerWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
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
    // An installed Worker: the one-command panel explains the local launcher against a new
    // invitation, shows no installation command (it was single-use) and keeps the files advanced (#277).
    const install = page.getByRole("region", { name: "One-command install" });
    await expect(install).toContainText("already installed");
    await expect(
      install.getByText(`snitch-worker/${firstId}/install.sh`),
    ).toBeVisible();
    await expect(
      page.getByRole("textbox", { name: "Worker installation command" }),
    ).toHaveCount(0);
    expect(await install.innerText()).not.toContain(firstToken);
    await install.getByText("Advanced: deployment files").click();
    // The token is never shown again: the file holds a placeholder.
    const env = page.getByRole("textbox", { name: "Content of .env" });
    expect(await env.inputValue()).not.toContain(firstToken);
    expect(await env.inputValue()).toContain(
      "SNITCH_WORKER_TOKEN=A_RENSEIGNER",
    );
    await expectNoViolations(page);

    // A lost token: confirmation, then the identity is replaced (new Worker, single-use command).
    await page
      .getByRole("button", { name: "Lost token: replace this Worker…" })
      .click();
    await expect(page.getByRole("alertdialog")).toContainText(
      "cannot be recovered",
    );
    await page.getByRole("button", { name: "Cancel" }).click();
    expect(await pulse(firstToken)).toBe(200);
    await page
      .getByRole("button", { name: "Lost token: replace this Worker…" })
      .click();
    await page
      .getByRole("checkbox", {
        name: /Allow an installation command over plain HTTP/,
      })
      .check();
    await page.getByRole("button", { name: "Replace", exact: true }).click();
    await expect(page.getByText("New Worker created.")).toBeVisible();
    const replacement = await page
      .getByRole("textbox", { name: "Worker installation command" })
      .inputValue();
    expect(replacement).toContain("/api/worker/install/");
    expect(await pulse(firstToken)).toBe(401);
    await page.getByRole("link", { name: "Open the new Worker" }).click();
    await expect(page).toHaveURL(/\/admin\/workers\/wk_/);
    const replacementId = page.url().split("/").pop()!;
    expect(replacementId).not.toBe(firstId);
    await expect(
      page.getByRole("heading", { name: "gaming-pc" }),
    ).toBeVisible();

    // Revocation: confirmation, then the Worker is revoked and its token refused.
    await page.getByRole("button", { name: "Delete this Worker…" }).click();
    await expect(page.getByRole("alertdialog")).toContainText(
      "Delete gaming-pc?",
    );
    await expectNoViolations(page);
    await page.getByRole("button", { name: "Delete", exact: true }).click();
    await expect(page.getByText(/This Worker was deleted on/)).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Delete this Worker…" }),
    ).toHaveCount(0);
    // A revoked Worker gets no recovery path: its identity is gone.
    await expect(
      page.getByRole("button", { name: "Diagnose / Reconnect" }),
    ).toHaveCount(0);
    expect(await pulse(firstToken)).toBe(401);

    // #261: the machine never confirmed, yet the Administrator can drop the entry after an
    // explicit warning that local cleanup is unconfirmed.
    await page.getByRole("button", { name: "Remove from the list" }).click();
    await expect(page.getByRole("alertdialog")).toContainText(
      "Local cleanup was not confirmed",
    );
    await page.getByRole("button", { name: "Cancel" }).click();
    await expect(page.getByRole("alertdialog")).toHaveCount(0);
    await page.getByRole("button", { name: "Remove from the list" }).click();
    await page.getByRole("button", { name: "Remove", exact: true }).click();
    await expect(page).toHaveURL(/\/admin\/workers$/);
    // The replaced identity keeps its revoked entry until it is forgotten too: target the
    // replacement by its address rather than by the shared "gaming-pc" name.
    await expect(
      page.locator(`a[href="/admin/workers/${replacementId}"]`),
    ).toHaveCount(0);
    await context.close();
  });

  test.describe("onglet Mises à jour", () => {
    const listing = {
      mode: "pinned",
      target_version: "v1.0.0",
      target_image: "ghcr.io/thomas-bsn/snitch-worker:v1.0.0",
      latest_version: "v1.1.0",
      checked_at: null,
      check_failed: false,
      image_repository: "ghcr.io/thomas-bsn/snitch-worker",
      catalog: ["v1.1.0", "v1.0.0"],
      catalog_available: true,
      workers: [],
    };

    async function open(browser: Browser, body?: () => unknown, status = 200) {
      const context = await browser.newContext({
        storageState: await sessionFor(ADMIN),
      });
      const page = await context.newPage();
      if (body) {
        await page.route("**/api/admin/worker-updates", (route) =>
          route.request().method() === "GET"
            ? route.fulfill({ status, json: body() })
            : route.continue(),
        );
      }
      await page.goto("/admin/worker-updates");
      return { context, page };
    }

    test("s'ouvre avec le formulaire et les Workers", async ({ browser }) => {
      const { context, page } = await open(browser, () => ({
        ...listing,
        workers: [
          {
            id: "wk_1",
            name: "gaming-pc",
            state: "updated",
            version: "v1.1.0",
            digest: "",
            cause_code: "",
            reported_at: null,
          },
        ],
      }));
      await expect(page.getByLabel("Target version")).toBeVisible();
      await expect(
        page.getByRole("button", { name: "Save target version" }),
      ).toBeVisible();
      await expect(page.getByRole("table")).toBeVisible();
      await expect(page.getByRole("cell", { name: "gaming-pc" })).toBeVisible();
      await context.close();
    });

    test("en mode Latest, la version manuelle est grisée et l'état du suivi est affiché", async ({
      browser,
    }) => {
      const { context, page } = await open(browser, () => ({
        ...listing,
        mode: "latest",
        target_version: "v1.1.0",
        checked_at: "2026-11-05T10:00:00Z",
        check_failed: true,
      }));
      await expect(page.getByLabel("Update strategy")).toHaveValue("latest");
      const version = page.getByLabel("Version applied to Workers");
      await expect(version).toBeDisabled();
      await expect(version).toHaveValue("v1.1.0");
      await expect(
        page.getByText("Latest stable version detected"),
      ).toBeVisible();
      await expect(
        page.getByText("The last registry check failed"),
      ).toBeVisible();
      // Back to a pinned version: the manual choice is editable again.
      await page.getByLabel("Update strategy").selectOption("pinned");
      await expect(page.getByLabel("Target version")).toBeEnabled();
      await context.close();
    });

    test("répond sans simulation sur la vraie API", async ({ browser }) => {
      const { context, page } = await open(browser);
      await expect(
        page.getByRole("button", { name: "Save target version" }),
      ).toBeVisible();
      await context.close();
    });

    test("un catalogue ou des Workers à null ne blanchissent pas la page", async ({
      browser,
    }) => {
      const { context, page } = await open(browser, () => ({
        ...listing,
        catalog: null,
        workers: null,
      }));
      await expect(page.getByRole("textbox")).toBeVisible();
      await expect(page.getByRole("navigation")).not.toHaveCount(0);
      await context.close();
    });

    test("un échec de l'API affiche une erreur", async ({ browser }) => {
      const { context, page } = await open(
        browser,
        () => ({ error: { code: "internal_error" } }),
        500,
      );
      await expect(page.getByRole("alert")).toBeVisible();
      await context.close();
    });

    test("une exception de rendu affiche un message, pas une page blanche", async ({
      browser,
    }) => {
      // A Worker without a name makes the row throw: the shell and a diagnostic must remain.
      const { context, page } = await open(browser, () => ({
        ...listing,
        workers: [null],
      }));
      await expect(page.getByRole("alert")).toContainText(
        "could not be displayed",
      );
      await expect(
        page.getByRole("navigation", { name: "Administration sections" }),
      ).toBeVisible();
      await context.close();
    });

    test("un membre ordinaire est refusé", async ({ browser }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(MEMBER),
      });
      const page = await context.newPage();
      await page.goto("/admin/worker-updates");
      await expect(
        page.getByRole("heading", { name: "Administrators only" }),
      ).toBeVisible();
      await context.close();
    });
  });
});
