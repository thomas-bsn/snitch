import AxeBuilder from "@axe-core/playwright";
import { test, expect, request, type Page } from "@playwright/test";
import {
  ADMIN,
  MEMBER,
  accountLanguage,
  createApprovedMember,
  installAdministrator,
  sessionFor,
} from "./shell-fixture";

// Account settings of the new interface: two tabs (Profile, My voice), the Discord link,
// the language kept in the account, and no theme setting (the theme lives in the profile
// menu). The deletion of a voice profile that has samples is covered, with a real
// diarized clip, by voice-profile.spec.ts.

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

/** The administrator enables Discord (simulated by the test server) through the API. */
async function enableDiscord() {
  const origin = process.env.SNITCH_TEST_URL!;
  const api = await request.newContext({
    baseURL: origin,
    extraHTTPHeaders: { Origin: origin },
    storageState: await sessionFor(ADMIN),
  });
  const me = await (await api.get("/api/auth/me")).json();
  const saved = await api.put("/api/admin/settings", {
    data: {
      client_id: "fixture-app",
      client_secret: "fixture-secret",
      registration_approval: false,
    },
    headers: { "X-CSRF-Token": me.csrf_token },
  });
  if (saved.status() !== 200) throw new Error("Discord not configured");
  await api.dispose();
}

test.describe.serial("paramètres du compte", () => {
  test.beforeAll(async () => {
    await installAdministrator();
    await createApprovedMember();
  });

  test("deux onglets adressables, et aucun réglage de thème", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await context.newPage();
    await page.goto("/account");
    await expect(
      page.getByRole("heading", { level: 1, name: "Account settings" }),
    ).toBeVisible();
    const tabs = page.getByRole("tablist", { name: "Account settings" });
    await expect(tabs.getByRole("tab")).toHaveText(["Profile", "My voice"]);
    await expect(tabs.getByRole("tab", { name: "Profile" })).toHaveAttribute(
      "aria-selected",
      "true",
    );

    // Each tab has its own address, which also works as an entry point.
    await tabs.getByRole("tab", { name: "My voice" }).click();
    await expect(page).toHaveURL(/\/account\/voice$/);
    await expect(
      page.getByRole("heading", { level: 2, name: "My voice", exact: true }),
    ).toBeVisible();
    await page.goBack();
    await expect(page).toHaveURL(/\/account$/);
    await expect(tabs.getByRole("tab", { name: "Profile" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    await page.goto("/account/voice");
    await expect(tabs.getByRole("tab", { name: "My voice" })).toHaveAttribute(
      "aria-selected",
      "true",
    );

    // The tabs answer the arrow keys.
    await tabs.getByRole("tab", { name: "My voice" }).focus();
    await page.keyboard.press("ArrowLeft");
    await expect(tabs.getByRole("tab", { name: "Profile" })).toBeFocused();

    // No theme setting here: the theme is chosen in the profile menu.
    for (const path of ["/account", "/account/voice"]) {
      await page.goto(path);
      const main = page.getByRole("main");
      await expect(main.getByText(/theme/i)).toHaveCount(0);
      await expect(main.getByRole("radio")).toHaveCount(0);
    }
    await context.close();
  });

  test("la langue se modifie depuis le profil et suit le compte", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await context.newPage();
    await page.goto("/account");
    await expect(page.getByRole("main").getByLabel("Language")).toHaveValue(
      "en",
    );
    await page.getByRole("main").getByLabel("Language").selectOption("fr");
    await expect(
      page.getByRole("heading", { level: 1, name: "Paramètres du compte" }),
    ).toBeVisible();
    await expect(page.getByRole("status")).toContainText("Langue enregistrée");
    await expect.poll(() => accountLanguage(MEMBER)).toBe("fr");

    // Remembered: the browser and another device both show it.
    await page.reload();
    await expect(
      page.getByRole("heading", { level: 1, name: "Paramètres du compte" }),
    ).toBeVisible();
    await expect(page.getByRole("main").getByLabel("Langue")).toHaveValue("fr");
    const device = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const devicePage = await device.newPage();
    await devicePage.goto("/account");
    await expect(
      devicePage.getByRole("heading", {
        level: 1,
        name: "Paramètres du compte",
      }),
    ).toBeVisible();

    await page.getByRole("main").getByLabel("Langue").selectOption("en");
    await expect(
      page.getByRole("heading", { level: 1, name: "Account settings" }),
    ).toBeVisible();
    await expect.poll(() => accountLanguage(MEMBER)).toBe("en");
    await device.close();
    await context.close();
  });

  test("sans Discord sur l'instance, le profil l'explique et ne propose rien", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await context.newPage();
    await page.goto("/account");
    await expect(page.getByRole("main")).toContainText(
      "Discord sign-in is not enabled",
    );
    await expect(
      page.getByRole("button", { name: "Link Discord" }),
    ).toHaveCount(0);
    await context.close();
  });

  test("ma voix sans Personne liée : explication et aucune suppression", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await context.newPage();
    await page.goto("/account/voice");
    await expect(page.getByRole("main")).toContainText(
      "No person is linked to your account",
    );
    await expect(
      page.getByRole("button", { name: "Delete my voice profile" }),
    ).toHaveCount(0);
    await context.close();
  });

  test("lier son identité Discord depuis le profil, avec confirmation et conflit signalé", async ({
    browser,
  }) => {
    await enableDiscord();
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await context.newPage();
    await page.goto("/account");
    await page.getByRole("button", { name: "Link Discord" }).click();
    await expect(
      page.getByRole("heading", { name: "Discord simulé" }),
    ).toBeVisible();
    await page
      .getByRole("link", { name: "Autoriser 654321", exact: true })
      .click();

    // The browser comes back to the profile with a confirmation that can be dismissed.
    await expect(page).toHaveURL(/\/account(\?|$)/);
    await expect(
      page.getByRole("status").filter({ hasText: "Your Discord identity is" }),
    ).toBeVisible();
    await expect(
      page.getByText(
        `Discord is linked to the Snitch account ${MEMBER.username}.`,
      ),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Unlink Discord" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Dismiss this message" }).click();
    await expect(page.getByRole("status")).toHaveCount(0);

    // The identity now opens this account.
    const visitorContext = await browser.newContext();
    const visitor = await visitorContext.newPage();
    await visitor.goto("/login");
    await visitor
      .getByRole("button", { name: "Continue with Discord" })
      .click();
    await visitor
      .getByRole("link", { name: "Autoriser 654321", exact: true })
      .click();
    await expect(
      visitor.getByRole("button", { name: "Account menu" }),
    ).toBeVisible();
    expect(
      await (
        await visitor.request.get(process.env.SNITCH_TEST_URL! + "/api/auth/me")
      ).json(),
    ).toMatchObject({ user: { username: MEMBER.username } });
    await visitorContext.close();

    // An identity that belongs to another account is refused and the link kept.
    const adminContext = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const admin = await adminContext.newPage();
    await admin.goto("/account");
    await admin.getByRole("button", { name: "Link Discord" }).click();
    await admin
      .getByRole("link", { name: "Autoriser 654321", exact: true })
      .click();
    await expect(
      admin.getByRole("status").filter({ hasText: "already linked" }),
    ).toBeVisible();
    await adminContext.close();

    await page.getByRole("button", { name: "Unlink Discord" }).click();
    const confirmation = page.getByRole("alertdialog", {
      name: "Unlink Discord?",
    });
    await expect(confirmation).toBeVisible();
    await confirmation
      .getByRole("button", { name: "Unlink Discord", exact: true })
      .click();
    await expect(page.getByRole("status")).toContainText(
      "Discord was unlinked.",
    );
    await expect(
      page.getByRole("button", { name: "Link Discord", exact: true }),
    ).toBeVisible();
    await context.close();
  });

  for (const theme of ["light", "dark"] as const) {
    test(`accessibilité des deux onglets, thème ${theme}`, async ({
      browser,
    }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(MEMBER),
      });
      await context.addInitScript((value) => {
        localStorage.setItem("snitch-ui-theme", value);
      }, theme);
      const page = await context.newPage();
      for (const path of ["/account", "/account/voice"]) {
        await page.goto(path);
        await expect(
          page.getByRole("heading", { level: 1, name: "Account settings" }),
        ).toBeVisible();
        await expectNoViolations(page);
      }
      await context.close();
    });
  }

  test("téléphone : les onglets et le formulaire tiennent à 390 pixels", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
      viewport: { width: 390, height: 780 },
    });
    const page = await context.newPage();
    await page.goto("/account");
    for (const tab of await page.getByRole("tab").all()) {
      expect((await tab.boundingBox())!.height).toBeGreaterThanOrEqual(44);
    }
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    await expectNoViolations(page);
    await context.close();
  });
});
