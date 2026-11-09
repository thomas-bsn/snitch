import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page, type Route } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// The page of a Worker shows its log since its last start and lets an Administrator pause or resume it.
// The Worker itself is not needed: the log and control endpoints are answered by the test, which also
// counts the requests, so that a polling that does not stop is caught.

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];
const line = (id: number, body: Record<string, unknown> | string) => ({
  id,
  seq: id - 1,
  at: "2026-11-09T17:20:03Z",
  line: typeof body === "string" ? body : JSON.stringify(body),
});

async function newWorker(page: Page): Promise<string> {
  const me = await (await page.request.get("/api/auth/me")).json();
  const created = await page.request.post("/api/admin/workers", {
    headers: {
      Origin: process.env.SNITCH_TEST_URL!,
      "X-CSRF-Token": me.csrf_token,
    },
    data: { name: `Synthetic logs ${Date.now()}`, max_concurrency: 1 },
  });
  expect(created.status()).toBe(201);
  return (await created.json()).id;
}

const json = (route: Route, body: unknown, status = 200) =>
  route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(body),
  });

test.describe.serial("Logs et contrôle d'un Worker", () => {
  test.beforeAll(async () => {
    await installAdministrator();
  });

  test("les logs depuis le démarrage, en direct, et plus anciens", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    const id = await newWorker(page);
    let afterRequests = 0;
    let beforeRequests = 0;
    const live = [line(4, { level: "INFO", event: "live_one" })];
    await page.route(`**/api/admin/workers/${id}/control`, (route) =>
      json(route, {
        paused: false,
        applied_paused: null,
      }),
    );
    await page.route(`**/api/admin/workers/${id}/logs**`, (route) => {
      const url = new URL(route.request().url());
      if (url.searchParams.has("before")) {
        beforeRequests++;
        return json(route, {
          session_id: "session-aaaa",
          has_more: false,
          lines: [line(1, { level: "INFO", event: "worker_started" })],
        });
      }
      if (url.searchParams.has("after")) {
        afterRequests++;
        return json(route, {
          session_id: "session-aaaa",
          has_more: false,
          lines: afterRequests === 1 ? live : [],
        });
      }
      return json(route, {
        session_id: "session-aaaa",
        has_more: true,
        lines: [
          line(2, {
            level: "ERROR",
            event: "engine_install_step_failed",
            step: "pip_check",
            exit_code: 1,
          }),
          // Not JSON, and shaped like markup: it must stay text.
          line(3, "<b>not a json line</b>"),
        ],
      });
    });

    await page.goto(`/admin/workers/${id}`);
    const region = page.getByRole("region", { name: "Worker log lines" });
    await expect(region).toContainText("engine_install_step_failed");
    await expect(region).toContainText("step=pip_check");
    await expect(region).toContainText("exit_code=1");
    await expect(region.getByText("Error", { exact: true })).toBeVisible();
    await expect(region.getByText("<b>not a json line</b>")).toBeVisible();
    await expect(region.locator("b")).toHaveCount(0);
    await expect(page.getByText("Current run · session-aaaa")).toBeVisible();

    // Live is on: the new line arrives without any click.
    await expect(region).toContainText("live_one", { timeout: 10_000 });

    // Older lines come on demand, and the button goes when the start of the run is reached.
    await page.getByRole("button", { name: "Load older lines" }).click();
    await expect(region).toContainText("worker_started");
    await expect(
      page.getByRole("button", { name: "Load older lines" }),
    ).toHaveCount(0);
    expect(beforeRequests).toBe(1);

    // Unchecked, nothing asks for lines any more.
    await page.getByLabel("Live").uncheck();
    const stopped = afterRequests;
    await page.waitForTimeout(5000);
    expect(afterRequests).toBe(stopped);
    await page.getByLabel("Live").check();
    await expect
      .poll(() => afterRequests, { timeout: 10_000 })
      .toBeGreaterThan(stopped);

    // Leaving the page stops it too.
    await page.goto("/admin/workers");
    const left = afterRequests;
    await page.waitForTimeout(5000);
    expect(afterRequests).toBe(left);

    await page.goto(`/admin/workers/${id}`);
    const { violations } = await new AxeBuilder({ page })
      .withTags(TAGS)
      .analyze();
    expect(violations.map((v) => v.id)).toEqual([]);
    await context.close();
  });

  test("pause et reprise, avec renvoi vers le redémarrage géré", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    const id = await newWorker(page);
    await page.route(`**/api/admin/workers/${id}/logs**`, (route) =>
      json(route, { session_id: "", has_more: false, lines: [] }),
    );
    let state = { paused: false, applied_paused: null as boolean | null };
    const posted: string[] = [];
    await page.route(`**/api/admin/workers/${id}/control`, async (route) => {
      if (route.request().method() === "POST") {
        const { action } = route.request().postDataJSON();
        posted.push(action);
        state = { ...state, paused: action === "pause" };
      }
      return json(route, state);
    });

    await page.goto(`/admin/workers/${id}`);
    await expect(page.getByText("No log received yet.")).toBeVisible();
    const status = (text: string) =>
      page.getByRole("status").filter({ hasText: text });
    await expect(status("Running")).toBeVisible();
    // No restart, stop or start here: the page points to the managed restart instead.
    await expect(
      page.getByRole("button", { name: /^(Restart|Stop|Start)$/ }),
    ).toHaveCount(0);
    await expect(
      page.getByText("It does not stop the container"),
    ).toBeVisible();
    await expect(
      page.getByText("enabled when the Worker is created"),
    ).toBeVisible();

    // Pause asks first, then waits for the Worker to confirm.
    await page.getByRole("button", { name: "Pause", exact: true }).click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toContainText("Jobs already running finish normally");
    await dialog.getByRole("button", { name: "Pause", exact: true }).click();
    await expect(status("Pausing…")).toBeVisible();
    state = { ...state, applied_paused: true };
    await expect(status("Paused")).toBeVisible({ timeout: 10_000 });

    await page.getByRole("button", { name: "Resume" }).click();
    expect(posted).toEqual(["pause", "resume"]);
    await expect(status("Resuming…")).toBeVisible();
    state = { ...state, applied_paused: false };
    await expect(status("Running")).toBeVisible({ timeout: 10_000 });
    await context.close();
  });
});
