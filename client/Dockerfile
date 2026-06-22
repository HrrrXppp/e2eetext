FROM node:22-alpine AS build

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# Serve static SPA only — no Node.js in the final image (~18 MB vs ~177 MB).
FROM ghcr.io/static-web-server/static-web-server:2-alpine

COPY --from=build /app/dist /public

EXPOSE 8080

CMD ["--root", "/public", "--page-fallback", "/public/index.html", "--host", "0.0.0.0", "--port", "8080", "--health", "true"]
