import { resolve } from "node:path";
import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";

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
      },
    },
  },
  server: {
    proxy: {
      "/api": process.env.SNITCH_API_PROXY_TARGET ?? "http://localhost:8080",
    },
  },
});
