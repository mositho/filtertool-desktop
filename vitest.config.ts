import { defineConfig } from "vitest/config"
import vue from "@vitejs/plugin-vue"
import { fileURLToPath, URL } from "node:url"

export default defineConfig({
  publicDir: fileURLToPath(new URL("./frontend/public", import.meta.url)),
  // Public URLs are served by Vite, not filesystem imports (especially on Windows).
  plugins: [vue({ template: { transformAssetUrls: { includeAbsolute: false } } })],
  resolve: { alias: { "@schema": fileURLToPath(new URL("./src/config", import.meta.url)) } },
  // Reuse jsdom across two workers while keeping each test file in its own VM context.
  test: { pool: "vmThreads", maxWorkers: 2, environment: "jsdom", include: ["test/**/*.test.ts"] },
})
