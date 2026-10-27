import { test, expect } from "@playwright/test";
import {
  ADMIN,
  MEMBER,
  approveSignUp,
  installAdministrator,
  sessionFor,
  setAccountLanguage,
} from "./shell-fixture";

const WRONG = "incorrect-password-1";

test.describe.serial("connexion dans la nouvelle interface", () => {
  test.beforeAll(installAdministrator);

  test("se connecter, voir l'erreur dans la langue choisie, se déconnecter", async ({
    page,
  }) => {
    await page.goto("/");
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();

    await page.getByLabel("Username").fill(ADMIN.username);
    await page.getByLabel("Password").fill(WRONG);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByRole("alert")).toHaveText(
      "Incorrect credentials, or the sign-up is still waiting for approval.",
    );

    // The same server code is shown in the language chosen on the page.
    await page.getByLabel("Language").selectOption("fr");
    await expect(
      page.getByRole("heading", { name: "Connexion" }),
    ).toBeVisible();
    await expect(page.getByRole("alert")).toHaveText(
      "Identifiants incorrects ou inscription en attente d’approbation.",
    );

    await page.getByLabel("Mot de passe").fill(ADMIN.password);
    await page.getByRole("button", { name: "Se connecter" }).click();
    await expect(page).toHaveURL(/\/library$/);
    await expect(
      page.getByRole("heading", { name: "Bibliothèque" }),
    ).toBeVisible();

    // The session survives a reload.
    await page.reload();
    await expect(
      page.getByRole("button", { name: "Menu du compte" }),
    ).toBeVisible();

    await page.getByRole("button", { name: "Menu du compte" }).click();
    await page.getByRole("menuitem", { name: "Se déconnecter" }).click();
    await expect(page).toHaveURL(/\/login$/);
    await expect(
      page.getByRole("heading", { name: "Connexion" }),
    ).toBeVisible();
    expect((await page.request.get("/api/auth/me")).status()).toBe(401);
    // The French chosen on the sign-in page was adopted by the account: the next journeys
    // run in English again.
    await setAccountLanguage(ADMIN, "en");
  });

  test("demander un compte, attendre l'approbation, puis se connecter", async ({
    page,
  }) => {
    await page.goto("/login");
    await page.getByRole("link", { name: "Create an account" }).click();
    await expect(
      page.getByRole("heading", { name: "Request an account" }),
    ).toBeVisible();
    await page.getByLabel("Username").fill(MEMBER.username);
    await page.getByLabel("Password").fill(MEMBER.password);
    await page.getByRole("button", { name: "Request access" }).click();
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByRole("status")).toContainText(
      "An administrator has to approve it",
    );

    // Not approved yet: the sign-in is refused with the same message as a wrong password.
    await page.getByLabel("Username").fill(MEMBER.username);
    await page.getByLabel("Password").fill(MEMBER.password);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByRole("alert")).toContainText(
      "still waiting for approval",
    );

    // A name already taken is reported in the page language.
    await page.getByRole("link", { name: "Create an account" }).click();
    await expect(
      page.getByRole("heading", { name: "Request an account" }),
    ).toBeVisible();
    await page.getByLabel("Username").fill(MEMBER.username);
    await page.getByLabel("Password").fill(MEMBER.password);
    await page.getByRole("button", { name: "Request access" }).click();
    await expect(page.getByRole("alert")).toHaveText(
      "This username is already taken.",
    );

    await approveSignUp(MEMBER);
    await page.getByRole("link", { name: "Back to sign in" }).click();
    await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
    await page.getByLabel("Username").fill(MEMBER.username);
    await page.getByLabel("Password").fill(MEMBER.password);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page).toHaveURL(/\/library$/);
  });

  test("l'administration n'apparaît que pour un administrateur", async ({
    browser,
  }) => {
    const member = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const memberPage = await member.newPage();
    await memberPage.goto("/library");
    await memberPage.getByRole("button", { name: "Account menu" }).click();
    await expect(memberPage.getByText("User", { exact: true })).toBeVisible();
    await expect(
      memberPage.getByRole("menuitem", { name: "Account settings" }),
    ).toBeVisible();
    await expect(
      memberPage.getByRole("menuitem", { name: "Administration" }),
    ).toHaveCount(0);
    await memberPage.keyboard.press("Escape");
    await memberPage.goto("/admin");
    await expect(
      memberPage.getByRole("heading", { name: "Administrators only" }),
    ).toBeVisible();
    await member.close();

    const admin = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const adminPage = await admin.newPage();
    await adminPage.goto("/library");
    await adminPage.getByRole("button", { name: "Account menu" }).click();
    await expect(
      adminPage.getByText("Super-administrator", { exact: true }),
    ).toBeVisible();
    await adminPage.getByRole("menuitem", { name: "Administration" }).click();
    await expect(adminPage).toHaveURL(/\/admin\/workers$/);
    await expect(
      adminPage.getByRole("heading", { name: "Administration" }),
    ).toBeVisible();
    await admin.close();
  });

  test("chaque écran a une adresse qui survit à l'actualisation", async ({
    page,
    browser,
  }) => {
    // Asking for a screen while signed out leads to the sign-in, then back to that screen.
    await page.goto("/people");
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
    await page.getByLabel("Username").fill(ADMIN.username);
    await page.getByLabel("Password").fill(ADMIN.password);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page).toHaveURL(/\/people$/);

    const screens = [
      ["/library", "Library", "/library"],
      ["/people", "People", "/people"],
      ["/import", "Import", "/import"],
      ["/account", "Account settings", "/account"],
      // Administration opens on its first tab.
      ["/admin", "Administration", "/admin/workers"],
    ] as const;
    for (const [path, title, landing] of screens) {
      await page.goto(path);
      await page.reload();
      await expect(page).toHaveURL(new RegExp(`${landing}$`));
      await expect(
        page.getByRole("heading", { name: title, level: 1 }),
      ).toBeVisible();
    }

    // The sidebar leads to the four screens and marks the current one.
    await page.goto("/library");
    const sidebar = page.getByRole("navigation", { name: "Main navigation" });
    await sidebar.getByRole("link", { name: "Search" }).click();
    await expect(page).toHaveURL(/\/search$/);
    await expect(sidebar.getByRole("link", { name: "Search" })).toHaveAttribute(
      "aria-current",
      "page",
    );

    // The header search opens the search screen with the query in the address.
    await page
      .getByRole("searchbox", { name: "Search the transcripts" })
      .fill("hello world");
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/search\?q=hello%20world$/);
    await page.reload();
    await expect(
      page.getByRole("searchbox", { name: "What was said" }),
    ).toHaveValue("hello world");

    // The Import button is always there.
    await page
      .getByRole("link", { name: "Import", exact: true })
      .first()
      .click();
    await expect(page).toHaveURL(/\/import$/);

    await page.goto("/nowhere/at-all");
    await expect(
      page.getByRole("heading", { name: "Page not found" }),
    ).toBeVisible();

    // A retired legacy URL serves the same single interface as the root URL.
    const visitor = await browser.newContext();
    const old = await visitor.newPage();
    await old.goto("/legacy/");
    await expect(old.getByRole("heading", { name: "Sign in" })).toBeVisible();
    expect(new URL(old.url()).pathname).toBe("/login");
    await visitor.close();
  });

  test("la nouvelle interface ne charge jamais l'ancienne feuille de style", async ({
    page,
  }) => {
    await page.goto("/login");
    await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
    const leaked = await page.evaluate(() =>
      [...document.styleSheets].some((sheet) =>
        [...sheet.cssRules].some(
          (rule) =>
            rule instanceof CSSStyleRule &&
            rule.selectorText.includes(".sidebar-user"),
        ),
      ),
    );
    expect(leaked).toBe(false);
  });

  test("un serveur injoignable affiche un message et « Try again » reprend", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/library");
    await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();

    const cut = (route: { abort: () => Promise<void> }) => route.abort();
    await page.route("**/api/**", cut);
    await page.reload();
    await expect(page.getByRole("alert")).toContainText("Can't reach Snitch");
    // Trying again while the server is still down keeps the message.
    await page.getByRole("button", { name: "Try again" }).click();
    await expect(page.getByRole("alert")).toContainText("Can't reach Snitch");

    await page.unroute("**/api/**", cut);
    await page.getByRole("button", { name: "Try again" }).click();
    await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
    await context.close();
  });
});
