import { resolve } from "node:path";
import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";

// Two entries: the new interface (index.html, the only one that loads Tailwind and the
// design tokens) and the old interface (legacy/index.html, served under /legacy/ during
// the transition, with its own global stylesheet). They share no module that carries CSS.
export default defineConfig({
  plugins: [tailwindcss()],
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
        legacy: resolve(import.meta.dirname, "legacy/index.html"),
      },
    },
  },
  server: { proxy: { "/api": "http://localhost:8080" } },
});
