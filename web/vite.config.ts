import { defineConfig } from "vite";
export default defineConfig({
  build: { outDir: "../app/ui/dist", emptyOutDir: true },
  server: { proxy: { "/api": "http://localhost:8080" } },
});
