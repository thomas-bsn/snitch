import { test } from "@playwright/test";

// Measurement of the Task Queue page on thousands of synthetic runs, not a regression test: it
// only runs when the Go measurement (SNITCH_PERF=1 SNITCH_PERF_PAGE=1) starts it against its own
// server and hands it a session. It logs what the page costs: DOM nodes, memory, long tasks.

const cookie = process.env.SNITCH_PERF_COOKIE ?? "";

test("the page of a queue of thousands of runs", async ({ browser }) => {
  test.skip(!cookie, "measurement only: started by the Go perf test");
  test.setTimeout(180_000);
  const [name, value] = cookie.split("=");
  const context = await browser.newContext({
    viewport: { width: 1600, height: 1000 },
  });
  await context.addCookies([
    { name, value, url: process.env.SNITCH_TEST_URL! },
  ]);
  const page = await context.newPage();
  await page.addInitScript(() => {
    const w = window as unknown as { __long: number[] };
    w.__long = [];
    new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) w.__long.push(entry.duration);
    }).observe({ entryTypes: ["longtask"] });
  });
  const started = Date.now();
  await page.goto("/admin/queue");
  await page.waitForSelector("[data-run-card]", { timeout: 120_000 });
  const firstCard = Date.now() - started;
  await page.waitForTimeout(3000);
  const cdp = await context.newCDPSession(page);
  await cdp.send("Performance.enable");
  const heap = async () => {
    await cdp.send("HeapProfiler.collectGarbage");
    const { metrics } = await cdp.send("Performance.getMetrics");
    return metrics.find((m) => m.name === "JSHeapUsedSize")!.value;
  };
  const shape = () =>
    page.evaluate(() => ({
      nodes: document.getElementsByTagName("*").length,
      cards: document.querySelectorAll("[data-run-card]").length,
    }));
  const first = await shape();
  const heapMb = (await heap()) / 1e6;
  const initialLong = await page.evaluate(
    () => (window as unknown as { __long: number[] }).__long,
  );
  // Updates pushed by the stream: a step changes every half second.
  await page.evaluate(() => {
    (window as unknown as { __long: number[] }).__long = [];
  });
  await page.waitForTimeout(4000);
  const streamLong = await page.evaluate(
    () => (window as unknown as { __long: number[] }).__long,
  );
  // An update: the stream is replaced by the polling fallback when the tab is shown again.
  await page.evaluate(() => {
    (window as unknown as { __long: number[] }).__long = [];
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await page.waitForTimeout(4000);
  const updateLong = await page.evaluate(
    () => (window as unknown as { __long: number[] }).__long,
  );
  const sum = (a: number[]) => Math.round(a.reduce((x, y) => x + y, 0));
  console.log(
    `PERF page: first_card_ms=${firstCard} dom_nodes=${first.nodes} run_cards=${first.cards} js_heap_mb=${heapMb.toFixed(1)} ` +
      `load_long_tasks=${initialLong.length} load_long_ms=${sum(initialLong)} load_max_ms=${Math.round(Math.max(0, ...initialLong))} ` +
      `stream_long_tasks=${streamLong.length} stream_long_ms=${sum(streamLong)} stream_max_ms=${Math.round(Math.max(0, ...streamLong))} refetch_long_tasks=${updateLong.length} refetch_long_ms=${sum(updateLong)} refetch_max_ms=${Math.round(Math.max(0, ...updateLong))}`,
  );
});
