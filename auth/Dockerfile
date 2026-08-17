FROM node:24-alpine AS build

WORKDIR /app

# Lockfile first so a source edit does not reinstall every dependency.
COPY package.json package-lock.json ./
RUN npm ci

COPY tsconfig.json ./
COPY src ./src
RUN npm run build

FROM node:24-alpine

WORKDIR /app
ENV NODE_ENV=production

COPY package.json package-lock.json ./
RUN npm ci --omit=dev && npm cache clean --force

COPY --from=build /app/dist ./dist

# node:alpine ships an unprivileged `node` user.
USER node

EXPOSE 3001

CMD ["node", "dist/server.js"]
