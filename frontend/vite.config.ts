import path from "node:path"
import { fileURLToPath } from "node:url"

import tailwindcss from "@tailwindcss/vite"
import { tanstackRouter } from "@tanstack/router-plugin/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

const root = path.dirname(fileURLToPath(import.meta.url))

const api = process.env.API_URL ?? "http://localhost:8080"
const auth = process.env.AUTH_URL ?? "http://localhost:3001"

export default defineConfig({
  plugins: [
    tanstackRouter({ target: "react", autoCodeSplitting: true }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": path.resolve(root, "./src"),
    },
  },
  server: {
    host: true,
    port: 5173,
    proxy: {
      // Browser stays on :5173 so the Better Auth cookie is same-origin.
      "/api/auth": { target: auth, changeOrigin: true },
      "/healthz": { target: api, changeOrigin: true },
      "/me": { target: api, changeOrigin: true },
      "/events": { target: api, changeOrigin: true },
      "/sessions": { target: api, changeOrigin: true },
    },
  },
})
