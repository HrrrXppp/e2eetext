import path from "node:path";
import { readFileSync } from "node:fs";
import type { Plugin, ProxyOptions } from "vite";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const ADSENSE_CLIENT = "ca-pub-8559868695202171";
const API_TARGET = "http://localhost:8080";
const appVersion = readFileSync(path.resolve(__dirname, "../VERSION"), "utf8").trim();

function apiProxyEntry(extra?: ProxyOptions): ProxyOptions {
  return {
    target: API_TARGET,
    changeOrigin: true,
    ...extra,
    configure: (proxy, options) => {
      extra?.configure?.(proxy, options);
      proxy.on("proxyReq", (proxyReq, req) => {
        const host = req.headers.host;
        if (typeof host === "string" && host) {
          proxyReq.setHeader("X-Forwarded-Host", host);
        }
        proxyReq.setHeader("X-Forwarded-Proto", "http");
      });
    },
  };
}

function adsensePlugin(enabled: boolean): Plugin {
  return {
    name: "adsense-production",
    transformIndexHtml(html) {
      if (!enabled) {
        return html;
      }

      return html.replace(
        "</head>",
        `    <script async src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=${ADSENSE_CLIENT}" crossorigin="anonymous"></script>\n  </head>`,
      );
    },
  };
}

const apiProxy = {
  "/api": apiProxyEntry({ secure: false, ws: true }),
  "/health": apiProxyEntry(),
};

export default defineConfig(({ mode }) => ({
  plugins: [react(), adsensePlugin(mode === "production")],
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
  },
}));
