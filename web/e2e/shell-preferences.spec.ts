import { test, expect } from "@playwright/test";
import {
  ADMIN,
  MEMBER,
  accountLanguage,
  createApprovedMember,
  installAdministrator,
  sessionFor,
} from "./shell-fixture";

test.describe.serial("langue et thème de la nouvelle interface", () => {
  test.beforeAll(async () => {
    await installAdministrator();
    await createApprovedMember();
  });

  test("l'anglais est la langue par défaut et rien n'est enregistré dans le compte", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await context.newPage();
    await page.goto("/library");
    await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
    await expect(page.locator("html")).toHaveAttribute("lang", "en");
    expect(await accountLanguage(MEMBER)).toBeNull();
    await context.close();
  });

  test("la langue d'un visiteur non connecté est retenue dans le navigateur", async ({
    page,
    browser,
  }) => {
    await page.goto("/login");
    await page.getByLabel("Language").selectOption("fr");
    await expect(
      page.getByRole("heading", { name: "Connexion" }),
    ).toBeVisible();
    await expect(page.locator("html")).toHaveAttribute("lang", "fr");
    await page.reload();
    await expect(
      page.getByRole("heading", { name: "Connexion" }),
    ).toBeVisible();
    await expect(page.getByLabel("Langue")).toHaveValue("fr");

    // Another browser knows nothing about it.
    const other = await browser.newContext();
    const otherPage = await other.newPage();
    await otherPage.goto("/login");
    await expect(
      otherPage.getByRole("heading", { name: "Sign in" }),
    ).toBeVisible();
    await other.close();
  });

  test("le choix fait avant de se connecter est adopté par un compte sans choix", async ({
    page,
  }) => {
    await page.goto("/login");
    await page.getByLabel("Language").selectOption("fr");
    await page.getByLabel("Nom d’utilisateur").fill(ADMIN.username);
    await page.getByLabel("Mot de passe").fill(ADMIN.password);
    await page.getByRole("button", { name: "Se connecter" }).click();
    await expect(
      page.getByRole("heading", { name: "Bibliothèque" }),
    ).toBeVisible();
    await expect.poll(() => accountLanguage(ADMIN)).toBe("fr");
  });

  test("la langue choisie une fois connecté est retenue dans le compte et suit l'appareil", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await context.newPage();
    await page.goto("/library");
    await page.getByRole("button", { name: "Account menu" }).click();
    await page.getByRole("menuitemradio", { name: "Français" }).click();
    await expect(
      page.getByRole("heading", { name: "Bibliothèque" }),
    ).toBeVisible();
    await expect.poll(() => accountLanguage(MEMBER)).toBe("fr");
    await context.close();

    // A device that never chose anything shows the account's language.
    const device = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const devicePage = await device.newPage();
    await devicePage.goto("/library");
    await expect(
      devicePage.getByRole("heading", { name: "Bibliothèque" }),
    ).toBeVisible();

    // And going back to English is saved the same way.
    await devicePage.getByRole("button", { name: "Menu du compte" }).click();
    await devicePage.getByRole("menuitemradio", { name: "English" }).click();
    await expect(
      devicePage.getByRole("heading", { name: "Library" }),
    ).toBeVisible();
    await expect.poll(() => accountLanguage(MEMBER)).toBe("en");
    await device.close();
  });

  test("sans choix enregistré, le thème suit le système, dès le premier affichage", async ({
    browser,
  }) => {
    for (const colorScheme of ["dark", "light"] as const) {
      const context = await browser.newContext({
        storageState: await sessionFor(MEMBER),
        colorScheme,
      });
      const page = await context.newPage();
      // Read before the application runs: the pre-paint script alone must get it right.
      const firstPaint: string[] = [];
      await page.exposeFunction("recordTheme", (value: string) =>
        firstPaint.push(value),
      );
      await page.addInitScript(() => {
        // The first theme set on <html>, i.e. by preferences-init.js before the app loads.
        const observer = new MutationObserver(() => {
          observer.disconnect();
          (
            window as unknown as { recordTheme: (value: string) => void }
          ).recordTheme(
            document.documentElement.getAttribute("data-theme") ?? "",
          );
        });
        observer.observe(document, {
          subtree: true,
          attributeFilter: ["data-theme"],
        });
      });
      await page.goto("/library");
      const theme = () => page.locator("html").getAttribute("data-theme");
      await expect.poll(theme).toBe(colorScheme);
      expect(firstPaint).toEqual([colorScheme]);
      // The account menu shows that the device is followed.
      await page
        .getByRole("button", { name: /Account menu|Menu du compte/ })
        .click();
      await expect(
        page.getByRole("menuitemradio", { name: "System" }),
      ).toBeChecked();
      await page.keyboard.press("Escape");
      // The device changing while the page is open is followed too.
      const other = colorScheme === "dark" ? "light" : "dark";
      await page.emulateMedia({ colorScheme: other });
      await expect.poll(theme).toBe(other);
      await context.close();
    }
  });

  test("un thème déjà enregistré reste respecté, quel que soit le système", async ({
    browser,
  }) => {
    for (const [stored, colorScheme] of [
      ["light", "dark"],
      ["dark", "light"],
    ] as const) {
      const context = await browser.newContext({
        storageState: await sessionFor(MEMBER),
        colorScheme,
      });
      await context.addInitScript((value) => {
        localStorage.setItem("snitch-ui-theme", value);
      }, stored);
      const page = await context.newPage();
      await page.goto("/library");
      await expect
        .poll(() => page.locator("html").getAttribute("data-theme"))
        .toBe(stored);
      await context.close();
    }
  });

  test("sombre, clair et système s'appliquent et sont retenus", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
      colorScheme: "light",
    });
    const page = await context.newPage();
    // The server forbids inline styles and scripts: nothing here may need them.
    const policyViolations: string[] = [];
    page.on("console", (message) => {
      if (message.text().includes("Content Security Policy"))
        policyViolations.push(message.text());
    });
    const theme = () => page.locator("html").getAttribute("data-theme");
    await page.goto("/library");
    await expect.poll(theme).toBe("light");

    const choose = async (name: string) => {
      await page
        .getByRole("button", { name: /Account menu|Menu du compte/ })
        .click();
      await page.getByRole("menuitemradio", { name }).click();
      await page.keyboard.press("Escape");
    };

    await choose("Dark");
    await expect.poll(theme).toBe("dark");
    await page.reload();
    await expect.poll(theme).toBe("dark");

    // "System" follows the device, including a change while the page is open.
    await choose("System");
    await expect.poll(theme).toBe("light");
    await page.emulateMedia({ colorScheme: "dark" });
    await expect.poll(theme).toBe("dark");
    await page.emulateMedia({ colorScheme: "light" });
    await expect.poll(theme).toBe("light");
    await page.emulateMedia({ colorScheme: "dark" });
    await page.reload();
    await expect.poll(theme).toBe("dark");

    // A fixed choice ignores the device again.
    await choose("Light");
    await expect.poll(theme).toBe("light");
    await page.reload();
    await expect.poll(theme).toBe("light");
    expect(policyViolations).toEqual([]);
    await context.close();
  });
});
