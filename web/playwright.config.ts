import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./e2e",
  workers: 1,
  retries: 0,
  use: {
    baseURL: process.env.SNITCH_TEST_URL ?? "http://localhost:8080",
    headless: true,
  },
  reporter: "list",
});
