import { defineConfig } from "vitest/config"
import vue from "@vitejs/plugin-vue"
import { fileURLToPath, URL } from "node:url"

export default defineConfig({
  plugins: [vue()],
  resolve: { alias: { "@schema": fileURLToPath(new URL("./src/config", import.meta.url)) } },
  test: { environment: "jsdom", include: ["test/**/*.test.ts"] },
})
