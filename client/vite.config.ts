import path from "node:path";
import { readFileSync } from "node:fs";
import type { ProxyOptions } from "vite";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const API_TARGET = "http://localhost:8080";
const appVersion = readFileSync(path.resolve(__dirname, "../VERSION"), "utf8").trim();

function apiProxyEntry(extra?: ProxyOptions): ProxyOptions {
  return {
    target: API_TARGET,
    changeOrigin: true,
    ...extra,
    configure: (proxy, options) => {
      extra?.configure?.(proxy, options);
      const forwardOriginHeaders = (proxyReq: { setHeader: (name: string, value: string) => void }, req: { headers: { host?: string } }) => {
        const host = req.headers.host;
        if (typeof host === "string" && host) {
          proxyReq.setHeader("X-Forwarded-Host", host);
        }
        proxyReq.setHeader("X-Forwarded-Proto", "http");
      };
      proxy.on("proxyReq", forwardOriginHeaders);
      // http-proxy fires a separate event for WebSocket upgrades — without
      // this, the server's Origin check (used for the /api/v1/ws upgrade)
      // sees the proxy target's Host instead of the browser's, and rejects
      // every WebSocket connection made through `npm run dev`.
      proxy.on("proxyReqWs", forwardOriginHeaders);
    },
  };
}

const apiProxy = {
  "/api": apiProxyEntry({ secure: false, ws: true }),
  "/health": apiProxyEntry(),
};

export default defineConfig({
  plugins: [react()],
  define: {
    "import.meta.env.VITE_APP_VERSION": JSON.stringify(appVersion),
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  server: {
    port: 5173,
    proxy: apiProxy,
  },
  preview: {
    proxy: apiProxy,
  },
  test: {
    environment: "jsdom",
    setupFiles: "./src/test/setup.ts",
    globals: true,
    // e2e/ holds Playwright specs (run via `npm run test:e2e`), not Vitest ones.
    exclude: ["**/node_modules/**", "e2e/**"],
  },
});
