# Production build only. Used by `make frontend-build` / CI.
# Dev serving is `make frontend` (Compose profile web), not `make up`.

FROM node:22-alpine
WORKDIR /app

COPY frontend/package.json frontend/package-lock.json frontend/.npmrc ./
RUN npm ci

COPY frontend ./
RUN npm run build
