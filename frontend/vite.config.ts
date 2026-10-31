import { defineConfig } from "vite"
import vue from "@vitejs/plugin-vue"
import tailwindcss from "@tailwindcss/vite"
import { fileURLToPath, URL } from "node:url"

const devPort = Number(process.env.WAILS_VITE_PORT || 5173)

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  root: fileURLToPath(new URL(".", import.meta.url)),
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
      "@schema": fileURLToPath(new URL("../src/config", import.meta.url)),
    },
  },
  server: {
    host: "127.0.0.1",
    port: devPort,
    strictPort: true,
    hmr: { host: "127.0.0.1", clientPort: devPort },
  },
})
