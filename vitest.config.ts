import { defineConfig } from "vitest/config"
import vue from "@vitejs/plugin-vue"
import { fileURLToPath, URL } from "node:url"

export default defineConfig({
  plugins: [vue()],
  resolve: { alias: { "@schema": fileURLToPath(new URL("./src/config", import.meta.url)) } },
  // Reuse jsdom across two workers while keeping each test file in its own VM context.
  test: { pool: "vmThreads", maxWorkers: 2, environment: "jsdom", include: ["test/**/*.test.ts"] },
})
