import { test, expect, request } from "@playwright/test";
import {
  ADMIN,
  MEMBER,
  approveSignUp,
  createApprovedMember,
  installAdministrator,
  sessionFor,
} from "./shell-fixture";

const OPERATOR = {
  username: "operator-user",
  password: "synthetic-password-789",
};

async function createOperator() {
  const origin = process.env.SNITCH_TEST_URL!;
  const guest = await request.newContext({
    baseURL: origin,
    extraHTTPHeaders: { Origin: origin },
  });
  expect(
    (await guest.post("/api/auth/register", { data: OPERATOR })).status(),
  ).toBe(201);
  await guest.dispose();
  await approveSignUp(OPERATOR);
  const api = await request.newContext({
    baseURL: origin,
    extraHTTPHeaders: { Origin: origin },
    storageState: await sessionFor(ADMIN),
  });
  const me = await (await api.get("/api/auth/me")).json();
  const { items: members }: { items: { id: number; username: string }[] } =
    await (await api.get("/api/admin/members")).json();
  const id = members.find((m) => m.username === OPERATOR.username)!.id;
  const done = await api.put(`/api/admin/members/${id}/role`, {
    data: { role: "admin" },
    headers: { "X-CSRF-Token": me.csrf_token },
  });
  expect(done.status()).toBe(200);
  await api.dispose();
}

test.describe.serial("file de tâches dans la barre latérale", () => {
  test.beforeAll(async () => {
    await installAdministrator();
    await createApprovedMember();
    await createOperator();
  });

  for (const [name, who] of [
    ["le super-administrateur", ADMIN],
    ["un administrateur", OPERATOR],
  ] as const) {
    test(`${name} voit l'entrée et ouvre la file`, async ({ browser }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(who),
        viewport: { width: 1440, height: 900 },
      });
      const page = await context.newPage();
      await page.goto("/library");
      const side = page.getByRole("navigation", { name: "Main navigation" });
      await side.getByRole("link", { name: "Task queue" }).first().click();
      await expect(page).toHaveURL(/\/queue$/);
      await expect(
        page.getByRole("heading", { level: 1, name: "Task queue" }),
      ).toBeVisible();
      // The tab is gone from the administration.
      await page.goto("/admin/workers");
      await expect(
        page
          .getByRole("navigation", { name: "Administration sections" })
          .getByRole("link", { name: "Task queue" }),
      ).toHaveCount(0);
      // The old address leads to the new one.
      await page.goto("/admin/queue");
      await expect(page).toHaveURL(/\/queue$/);
      await context.close();
    });
  }

  test("un membre ne voit pas l'entrée et reçoit un refus", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await context.newPage();
    await page.goto("/library");
    await expect(page.getByRole("link", { name: "Task queue" })).toHaveCount(0);
    await page.goto("/queue");
    await expect(
      page.getByRole("heading", { name: "Administrators only" }),
    ).toBeVisible();
    await page.goto("/admin/queue");
    await expect(
      page.getByRole("heading", { name: "Administrators only" }),
    ).toBeVisible();
    await context.close();
  });

  test("à 390 px, la barre du bas reste lisible pour un administrateur", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
      viewport: { width: 390, height: 780 },
    });
    const page = await context.newPage();
    await page.goto("/library");
    const tabs = page.getByRole("navigation", { name: "Main navigation" });
    await expect(tabs.getByRole("link")).toHaveCount(4);
    for (const link of await tabs.getByRole("link").all()) {
      const box = await link.boundingBox();
      expect(box!.height).toBeGreaterThanOrEqual(44);
      expect(box!.width).toBeGreaterThanOrEqual(44);
      const truncated = await link.evaluate(
        (el) => el.scrollWidth > el.clientWidth,
      );
      expect(truncated).toBe(false);
    }
    await context.close();
  });

  test("captures de la barre latérale", async ({ browser }) => {
    test.skip(!process.env.SNITCH_SHOTS, "captures à la demande");
    for (const theme of ["light", "dark"]) {
      for (const [w, h] of [
        [1440, 900],
        [390, 780],
      ]) {
        const context = await browser.newContext({
          storageState: await sessionFor(ADMIN),
          viewport: { width: w, height: h },
        });
        await context.addInitScript((value) => {
          localStorage.setItem("snitch-ui-theme", value);
        }, theme);
        const page = await context.newPage();
        await page.goto("/queue");
        await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
        await page.screenshot({
          path: `${process.env.SNITCH_SHOTS}/queue-nav-${theme}-${w}.png`,
        });
        await context.close();
      }
    }
  });
});
