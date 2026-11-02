const { chromium } = require("@playwright/test");
(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({
    viewport: { width: 1440, height: 1000 },
  });
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.route("**/api/**", async (route) => {
    const u = new URL(route.request().url());
    if (!u.pathname.startsWith("/api/")) return route.continue();
    let data = {};
    const job = (id) => ({
      id,
      clip_id: id,
      step: "transcription.audio.1",
      status: u.searchParams.get("workflow") === "done" ? "done" : "leased",
      updated_at: "2026-10-30T00:00:00Z",
      last_error: "",
    });
    if (u.pathname === "/api/setup")
      data = { initialized: true, discord_enabled: false };
    else if (u.pathname === "/api/auth/me")
      data = {
        user: {
          id: 1,
          username: "Fixture",
          role: "super_admin",
          status: "active",
          language: "en",
        },
        csrf_token: "fixture",
      };
    else if (u.pathname === "/api/admin/job-controls")
      data = { paused: false, max_concurrency: 4 };
    else if (u.pathname === "/api/admin/workers")
      data = {
        items: [
          {
            id: "fixture",
            name: "Fixture",
            supported_steps: ["transcription.audio.*", "voice.*"],
            revoked_at: null,
          },
        ],
      };
    else if (u.pathname === "/api/admin/analysis/queue")
      data = { active: [], completed: [], tasks: [] };
    else if (u.pathname === "/api/admin/analysis/history")
      data = {
        items: [],
        total: 25,
        page: Number(u.searchParams.get("page") || 1),
        page_size: 10,
      };
    else if (u.pathname === "/api/admin/jobs") {
      const p = Number(u.searchParams.get("page") || 1);
      data = {
        items: [
          job(u.searchParams.get("workflow") === "done" ? 100 + p : p + 1),
        ],
        page: p,
        page_size: 10,
        total: 25,
      };
    } else if (u.pathname.startsWith("/api/clips/"))
      data = { original_name: "Fixture video.mp4" };
    else if (u.pathname === "/api/admin/setup-progress")
      data = { dismissed: true, steps: [] };
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(data),
    });
  });
  await page.goto("http://127.0.0.1:5176/admin/queue");
  await page
    .getByRole("button", { name: "Clip history", exact: true })
    .waitFor();
  await page.getByRole("button", { name: "Clip history", exact: true }).click();
  await page.getByRole("dialog").waitFor();
  await page.getByRole("button", { name: "Next", exact: true }).last().click();
  await page.waitForTimeout(300);
  if (!(await page.getByText("Page 2 · 25 analyses").isVisible()))
    throw Error("history page missing");
  await page.getByRole("button", { name: "Close", exact: true }).click();
  // The advanced Schedule form is open by default: clicking its summary would close it.
  const select = page.locator('select[name="step"]');
  await select.waitFor({ state: "visible" });
  if ((await select.locator("option").count()) !== 3)
    throw Error("worker capabilities not reflected");
  await page.setViewportSize({ width: 390, height: 844 });
  if (
    await page.evaluate(
      () => document.documentElement.scrollWidth > window.innerWidth,
    )
  )
    throw Error("mobile overflow");
  console.log(
    "PASS: authenticated queue, history page 2, modal close, worker step choices, mobile overflow, no JS errors",
  );
  if (errors.length) throw Error(errors.join("\n"));
  await page.screenshot({
    path: "/tmp/snitch-queue-desktop.png",
    fullPage: true,
  });
  await browser.close();
})();
