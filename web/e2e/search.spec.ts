import { expect, test } from "@playwright/test";

test("recherche exacte française, filtres et lecture du passage", async ({ page }) => {
  await page.goto("/");
  await page.getByLabel("Code d’installation").fill(process.env.RECEIPTS_TEST_CODE!);
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page.getByLabel("Mot de passe", { exact: true }).fill("synthetic-password-123");
  await page.getByRole("button", { name: "Créer l’administrateur" }).click();
  await expect(page.getByRole("heading", { name: "Bibliothèque", exact: true })).toBeVisible();

  const fixture = await page.evaluate(async () => {
    const me = await fetch("/api/auth/me", { credentials: "same-origin" }).then((response) => response.json());
    const headers = { "Content-Type": "application/json", "X-CSRF-Token": me.csrf_token, Origin: location.origin };
    const bytes = new TextEncoder().encode("synthetic-search-video");
    const uploadResponse = await fetch("/api/uploads", { method: "POST", headers, body: JSON.stringify({ filename: "synthetic-search.mp4", size_bytes: bytes.length, visibility: "private" }) });
    if (!uploadResponse.ok) throw new Error(`upload creation ${uploadResponse.status}: ${await uploadResponse.text()}`);
    const upload = await uploadResponse.json();
    await fetch(`/api/uploads/${upload.id}/chunks`, { method: "PUT", headers: { ...headers, "Content-Type": "application/octet-stream", "Content-Range": `bytes 0-${bytes.length - 1}/${bytes.length}` }, body: bytes });
    const completedResponse = await fetch(`/api/uploads/${upload.id}/complete`, { method: "POST", headers, body: "{}" });
    if (!completedResponse.ok) throw new Error(`upload completion ${completedResponse.status}: ${await completedResponse.text()}`);
    const completed = await completedResponse.json();
    const selected = await fetch(`/api/admin/clips/${completed.clip_id}/audio-track`, { method: "PUT", headers, body: JSON.stringify({ audio_stream_index: 2 }) });
    if (!selected.ok) throw new Error(`track selection ${selected.status}: ${await selected.text()}`);
    const scheduled = await fetch(`/api/admin/clips/${completed.clip_id}/transcription`, { method: "POST", headers, body: "{}" });
    if (!scheduled.ok) throw new Error(`transcription schedule ${scheduled.status}: ${await scheduled.text()}`);
    const worker = await fetch("/api/admin/workers", { method: "POST", headers, body: JSON.stringify({ name: "synthetic-search-worker", max_concurrency: 1 }) }).then((response) => response.json());
    const workerCall = (path: string, body?: unknown, lease?: string) => fetch(path, { method: "POST", headers: { Authorization: `Bearer ${worker.token}`, ...(body ? { "Content-Type": "application/json" } : {}), ...(lease ? { "X-Worker-Lease-Token": lease } : {}) }, body: body ? JSON.stringify(body) : undefined });
    await workerCall("/api/worker/pulse", { capacity: 1, supported_steps: ["transcription.audio.*"] });
    const assignment = await workerCall("/api/worker/jobs/claim").then((response) => response.json());
    if (!assignment.job) throw new Error(`no transcription job was claimed: ${JSON.stringify(assignment)}`);
    const completion = await workerCall(`/api/worker/jobs/${assignment.job.id}/complete`, { input_version: assignment.job.input_version, result: { provider: "fixture", model: "deterministic-v1", version: "deterministic-v1", language: "fr", audio_stream_index: 2, segments: [{ start: 1.25, end: 2.1, text: "Bien joué !", words: [{ word: "Bien", start: 1.25, end: 1.52 }, { word: "joué", start: 1.55, end: 1.9 }] }] } }, assignment.lease_token);
    const search = await fetch("/api/search?q=JOUE&mode=exact&page=1&page_size=10").then((response) => response.json());
    const fullText = await fetch("/api/search?q=JO&mode=full_text&visibility=private&status=ready&from=2025-01-01&to=2025-01-03").then((response) => response.json());
    const approximate = await fetch("/api/search?q=jouee&mode=full_text&visibility=private&status=ready&from=2025-01-01&to=2025-01-03").then((response) => response.json());
    const sourceFilter = await fetch("/api/search?q=JOUE&mode=exact&source_id=1").then((response) => response.json());
    const unrelated = await fetch("/api/search?q=zzzqqq&mode=full_text").then((response) => response.json());
    return { clip_id: completed.clip_id, completion: completion.status, search, fullText, approximate, sourceFilter, unrelated };
  });
  expect(fixture.completion).toBe(200);
  expect(fixture.search).toMatchObject({ total: 1, page: 1, items: [{ clip_id: fixture.clip_id, match_type: "exact", timestamp: 1.55, matched_text: "joué" }] });
  expect(fixture.fullText).toMatchObject({ total: 1, items: [{ clip_id: fixture.clip_id, match_type: "full_text", timestamp: 1.55 }] });
  expect(fixture.approximate).toMatchObject({ total: 1, items: [{ clip_id: fixture.clip_id, match_type: "approximate", timestamp: 1.55, matched_text: "joué" }] });
  expect(fixture.unrelated).toMatchObject({ total: 0, items: [] });
  expect(fixture.sourceFilter.total).toBe(0);

  await page.getByLabel("Rechercher dans les paroles").fill("JOUE");
  await page.getByLabel("Type de recherche").selectOption("exact");
  await page.getByLabel("Visibilité de recherche").selectOption("private");
  await page.getByRole("button", { name: "Rechercher", exact: true }).click();
  const search = page.getByRole("region", { name: "Recherche dans les paroles" });
  await expect(search.getByText("1 résultat · page 1")).toBeVisible();
  await expect(search.locator("mark")).toHaveText("joué");
  await page.getByLabel("Rechercher dans les paroles").fill("jouee");
  await page.getByLabel("Type de recherche").selectOption("full_text");
  await page.getByRole("button", { name: "Rechercher", exact: true }).click();
  await expect(search.getByText("Correspondance approximative")).toBeVisible();
  await expect(search.locator("mark")).toHaveText("joué");
  await page.getByLabel("Rechercher dans les paroles").fill("JOUE");
  await page.getByLabel("Type de recherche").selectOption("exact");
  await page.getByRole("button", { name: "Rechercher", exact: true }).click();
  await search.getByRole("button", { name: "Ouvrir à 00:01" }).click();
  await expect(search.getByLabel("Lecture synthetic-search.mp4")).toBeVisible();
  await page.evaluate(() => {
    const player = document.querySelector(".search-playback video") as HTMLVideoElement;
    Object.defineProperty(player, "currentTime", { configurable: true, set: (value) => { (window as Window & { searchTime?: number }).searchTime = value; } });
    player.dispatchEvent(new Event("loadedmetadata"));
  });
  await expect.poll(() => page.evaluate(() => (window as Window & { searchTime?: number }).searchTime)).toBe(1.55);
  await expect(page.getByLabel("Rechercher dans les paroles")).toHaveValue("JOUE");

  const privateAccess = await page.evaluate(async (clipID: number) => {
    const headers = (csrf: string) => ({ "Content-Type": "application/json", "X-CSRF-Token": csrf, Origin: location.origin });
    const me = await fetch("/api/auth/me").then((response) => response.json());
    const registered = await fetch("/api/auth/register", { method: "POST", headers: { "Content-Type": "application/json", Origin: location.origin }, body: JSON.stringify({ username: "search-reader", password: "synthetic-password-456" }) });
    if (!registered.ok) throw new Error(`member registration ${registered.status}: ${await registered.text()}`);
    const pending = await fetch("/api/admin/users").then((response) => response.json());
    const user = pending.find((item: { username: string }) => item.username === "search-reader");
    if (!user) throw new Error("registered search reader was not pending");
    await fetch(`/api/admin/users/${user.id}/approve`, { method: "POST", headers: headers(me.csrf_token), body: "{}" });
    await fetch("/api/auth/logout", { method: "POST", headers: headers(me.csrf_token), body: "{}" });
    const login = await fetch("/api/auth/login", { method: "POST", headers: { "Content-Type": "application/json", Origin: location.origin }, body: JSON.stringify({ username: "search-reader", password: "synthetic-password-456" }) });
    if (!login.ok) throw new Error(`member login ${login.status}: ${await login.text()}`);
    const [search, detail, media] = await Promise.all([
      fetch("/api/search?q=JOUE&mode=exact").then((response) => response.json()),
      fetch(`/api/clips/${clipID}`).then((response) => response.status),
      fetch(`/api/clips/${clipID}/media`).then((response) => response.status),
    ]);
    return { total: search.total, detail, media };
  }, fixture.clip_id);
  expect(privateAccess).toEqual({ total: 0, detail: 404, media: 404 });
});
