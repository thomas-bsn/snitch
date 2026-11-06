import { expect, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// The Workers list and the page of a Worker follow the server's events: a Worker that is created,
// connects or goes silent is shown without reloading. Requests to the same resources are counted:
// while the stream is up, the pages must not ask for a snapshot themselves.

const NAME = "Synthetic live worker";

test.describe.serial("Workers en direct", () => {
  test.beforeAll(async () => {
    await installAdministrator();
  });

  test("la liste et la fiche changent d'état sans rechargement", async ({
    browser,
  }) => {
    // Going Offline takes the connected window (30 s) plus a check: longer than the default timeout.
    test.setTimeout(120_000);
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const list = await context.newPage();
    await list.goto("/admin/workers");
    await list.evaluate(() => {
      (window as unknown as Record<string, unknown>).marker = "kept";
    });
    const row = list.locator("tr").filter({ hasText: NAME });

    // Created elsewhere: the list shows it at once.
    const me = await (await list.request.get("/api/auth/me")).json();
    const headers = {
      Origin: process.env.SNITCH_TEST_URL!,
      "X-CSRF-Token": me.csrf_token,
    };
    const created = await list.request.post("/api/admin/workers", {
      headers,
      data: { name: NAME, max_concurrency: 1 },
    });
    expect(created.status()).toBe(201);
    const worker = await created.json();
    await expect(row).toContainText("Offline", { timeout: 10_000 });

    // The page of the Worker is open next to the list.
    const detail = await context.newPage();
    await detail.goto(`/admin/workers/${worker.id}`);
    await expect(detail.getByText("Offline", { exact: true })).toBeVisible();
    await expect(
      detail.getByRole("button", { name: "Diagnose / Reconnect" }),
    ).toBeVisible();
    await list.waitForTimeout(1000);

    const polls: string[] = [];
    const watch = (request: { url(): string; method(): string }) => {
      const { pathname } = new URL(request.url());
      if (
        request.method() === "GET" &&
        (pathname === "/api/admin/workers" ||
          pathname === `/api/admin/workers/${worker.id}`)
      ) {
        polls.push(pathname);
      }
    };
    list.on("request", watch);
    detail.on("request", watch);

    // The Worker connects.
    const connected = await list.request.post("/api/worker/pulse", {
      headers: { Authorization: `Bearer ${worker.token}` },
      data: { capacity: 1, supported_steps: ["fixture.*"] },
    });
    expect(connected.status()).toBe(200);
    await expect(row).toContainText("Online", { timeout: 10_000 });
    await expect(detail.getByText("Online", { exact: true })).toBeVisible({
      timeout: 10_000,
    });

    // It goes silent: no write announces it, the server does.
    await expect(row).toContainText("Offline", { timeout: 60_000 });
    await expect(detail.getByText("Offline", { exact: true })).toBeVisible({
      timeout: 10_000,
    });

    // Installed once, offline now: both are said, apart, and the recovery leads are offered (#263).
    await expect(
      detail.getByRole("region", { name: "Installation" }),
    ).toContainText("Complete");
    const toggle = detail.getByRole("button", { name: "Diagnose / Reconnect" });
    await toggle.focus();
    await detail.keyboard.press("Enter");
    await expect(toggle).toHaveAttribute("aria-expanded", "true");
    const diagnosis = detail.getByRole("region", {
      name: "Connection and recovery",
    });
    await expect(
      diagnosis.getByText(`docker restart snitch-worker-${worker.id}`),
    ).toBeVisible();
    // No remote restart is claimed, and no secret is shown.
    await expect(diagnosis).toContainText(
      "cannot restart this machine remotely",
    );
    await expect(diagnosis).toContainText("Waiting for a new pulse");
    await expect(diagnosis).not.toContainText("The Worker is back online");
    expect(await diagnosis.innerText()).not.toContain(worker.token);
    expect(await diagnosis.innerHTML()).not.toContain(worker.token);

    // Only a new pulse announces the return.
    const again = await list.request.post("/api/worker/pulse", {
      headers: { Authorization: `Bearer ${worker.token}` },
      data: { capacity: 1, supported_steps: ["fixture.*"] },
    });
    expect(again.status()).toBe(200);
    await expect(diagnosis).toContainText("The Worker is back online", {
      timeout: 10_000,
    });

    expect(polls).toEqual([]);
    expect(
      await list.evaluate(
        () => (window as unknown as Record<string, unknown>).marker,
      ),
    ).toBe("kept");
    await context.close();
  });
});
