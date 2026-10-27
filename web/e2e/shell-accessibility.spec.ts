import AxeBuilder from "@axe-core/playwright";
import { test, expect, type Page } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";
import { uploadClip } from "./library-fixture";

// Automatic accessibility check (WCAG 2.2 AA rules) on the screens of the new interface,
// in both themes and at phone width.

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];

async function expectNoViolations(page: Page) {
  const { violations } = await new AxeBuilder({ page })
    .withTags(TAGS)
    .analyze();
  expect(
    violations.map(
      (v) => `${v.id}: ${v.nodes.map((n) => n.target.join(" ")).join(" | ")}`,
    ),
    new URL(page.url()).pathname,
  ).toEqual([]);
}

async function expectTouchTargets(page: Page) {
  const undersized = await page
    .locator(
      'button, a, input, select, textarea, [role="button"], [role="tab"]',
    )
    .evaluateAll((elements) =>
      elements.flatMap((element) => {
        if (element.classList.contains("sr-only")) {
          return [];
        }
        let target = element;
        if (
          element.tagName === "INPUT" &&
          ["checkbox", "radio"].includes(element.getAttribute("type") ?? "")
        ) {
          target = element.closest("label") ?? element;
        } else if (element.classList.contains("after:absolute")) {
          let ancestor = element.parentElement;
          while (ancestor && getComputedStyle(ancestor).position === "static") {
            ancestor = ancestor.parentElement;
          }
          target = ancestor ?? element;
        }
        const box = target.getBoundingClientRect();
        if (box.width === 0 || box.height === 0) return [];
        return box.width < 44 || box.height < 44
          ? [
              `${element.tagName.toLowerCase()} ${element.getAttribute("type") ?? ""} ${element.getAttribute("aria-label") ?? ""} ${Math.round(box.width)}x${Math.round(box.height)}`,
            ]
          : [];
      }),
    );
  expect(undersized, new URL(page.url()).pathname).toEqual([]);
}

test.describe.serial("accessibilité de la nouvelle interface", () => {
  test.beforeAll(installAdministrator);

  const clipPage = async () =>
    `/clips/${await uploadClip("synthetic-accessibility.mp4", "shared")}`;

  for (const theme of ["light", "dark"] as const) {
    test(`écrans de connexion, thème ${theme}`, async ({ browser }) => {
      const context = await browser.newContext();
      await context.addInitScript((value) => {
        localStorage.setItem("snitch-ui-theme", value);
      }, theme);
      const page = await context.newPage();
      await page.goto("/login");
      await expect(
        page.getByRole("heading", { name: "Sign in" }),
      ).toBeVisible();
      await expectNoViolations(page);
      await page.getByRole("link", { name: "Create an account" }).click();
      await expect(
        page.getByRole("heading", { name: "Request an account" }),
      ).toBeVisible();
      await expectNoViolations(page);
      await page.getByRole("link", { name: "Back to sign in" }).click();
      await expect(
        page.getByRole("heading", { name: "Sign in" }),
      ).toBeVisible();
      await page.getByLabel("Username").fill(ADMIN.username);
      await page.getByLabel("Password").fill("incorrect-password-1");
      await page.getByRole("button", { name: "Sign in" }).click();
      await expect(page.getByRole("alert")).toBeVisible();
      await expectNoViolations(page);
      await context.close();
    });

    test(`coque, menu du profil et écrans, thème ${theme}`, async ({
      browser,
    }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(ADMIN),
      });
      await context.addInitScript((value) => {
        localStorage.setItem("snitch-ui-theme", value);
      }, theme);
      const page = await context.newPage();
      const paths = [
        "/library",
        "/search?q=test",
        "/people",
        "/people/verification",
        "/people/groups",
        "/import",
        "/account",
        "/account/profile",
        "/account/voice",
        "/admin",
        "/admin/workers",
        "/admin/workers/999999",
        "/admin/queue",
        "/admin/folders",
        "/admin/transcription",
        "/admin/transcription/vocabulary",
        "/admin/voice",
        "/admin/semantic",
        "/admin/users",
        "/admin/users/signups",
        "/admin/exports",
        await clipPage(),
        "/missing",
      ];
      for (const path of paths) {
        await page.goto(path);
        await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
        await expectNoViolations(page);
      }
      await page.goto("/library");
      await page.getByRole("button", { name: "Account menu" }).click();
      await expect(
        page.getByRole("menuitem", { name: "Administration" }),
      ).toBeVisible();
      await expectNoViolations(page);
      // The menu closes with Escape and gives the focus back to its button.
      await page.keyboard.press("Escape");
      await expect(page.getByRole("menu")).toHaveCount(0);
      await expect(
        page.getByRole("button", { name: "Account menu" }),
      ).toBeFocused();
      await context.close();
    });
  }

  test("téléphone : barre d'onglets en bas à la place de la barre latérale", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
      viewport: { width: 390, height: 780 },
    });
    const page = await context.newPage();
    const paths = [
      "/library",
      "/search?q=test",
      "/people",
      "/people/verification",
      "/people/groups",
      "/import",
      "/account",
      "/account/profile",
      "/account/voice",
      "/admin/workers",
      "/admin/workers/999999",
      "/admin/queue",
      "/admin/folders",
      "/admin/transcription",
      "/admin/transcription/vocabulary",
      "/admin/voice",
      "/admin/semantic",
      "/admin/users",
      "/admin/users/signups",
      "/admin/exports",
      await clipPage(),
    ];
    for (const path of paths) {
      await page.goto(path);
      await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
      await expectNoViolations(page);
      await expectTouchTargets(page);
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= window.innerWidth,
        ),
      ).toBe(true);
    }

    await page.goto("/library");
    const tabs = page.getByRole("navigation", { name: "Main navigation" });
    await expect(tabs).toBeVisible();
    await expect(tabs.getByRole("link")).toHaveCount(4);
    for (const link of await tabs.getByRole("link").all()) {
      const box = await link.boundingBox();
      expect(box!.height).toBeGreaterThanOrEqual(44);
    }
    const avatar = await page
      .getByRole("button", { name: "Account menu" })
      .boundingBox();
    expect(avatar!.width).toBeGreaterThanOrEqual(44);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    await expectNoViolations(page);
    await context.close();
  });
});
