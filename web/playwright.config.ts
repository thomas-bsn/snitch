import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./e2e",
  workers: 1,
  retries: 0,
  use: {
    baseURL: process.env.SNITCH_TEST_URL ?? "http://localhost:8080",
    headless: true,
    actionTimeout: 15000,
    // The theme follows the device by default: pin it so screens render the same everywhere.
    colorScheme: "light",
  },
  reporter: "list",
});
