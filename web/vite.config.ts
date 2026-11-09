import { writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";

// emptyOutDir wipes the tracked app/ui/dist/.gitkeep (the directory must exist for go:embed on a fresh clone):
// put it back once the bundle is written, so a build leaves `git status` clean.
const keepTrackedPlaceholder = {
  name: "keep-tracked-placeholder",
  apply: "build" as const,
  closeBundle() {
    writeFileSync(resolve(import.meta.dirname, "../app/ui/dist/.gitkeep"), "");
  },
};

export default defineConfig({
  plugins: [tailwindcss(), keepTrackedPlaceholder],
  build: {
    outDir: "../app/ui/dist",
    emptyOutDir: true,
    rollupOptions: {
      // Radix and the icon library mark modules "use client" (a React Server Components
      // hint with no meaning in a browser-only bundle): the warning is noise.
      onwarn(warning, defaultHandler) {
        if (warning.code === "MODULE_LEVEL_DIRECTIVE") return;
        defaultHandler(warning);
      },
      input: {
        main: resolve(import.meta.dirname, "index.html"),
      },
    },
  },
  server: {
    proxy: {
      "/api": process.env.SNITCH_API_PROXY_TARGET ?? "http://localhost:8080",
    },
  },
});
