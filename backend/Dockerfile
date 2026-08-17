# Builds every command in ./cmd into one small image. Which binary runs is chosen by the
# compose service, so the migrator and the API server share this file and its layer cache.

FROM golang:1.26-alpine AS build

WORKDIR /src

# Dependencies first, in their own layer, so editing Go source does not re-download the
# module cache on every build.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

# Keep this stage before the runtime image so `docker compose build` without a
# target still produces the small api/migrate/seed image.
FROM build AS test
CMD ["go", "test", "./..."]

FROM alpine:3.22

# Not root, and no shell access needed to run a compiled binary.
RUN adduser -D -u 10001 app

COPY --from=build /out/ /usr/local/bin/

USER app

ENTRYPOINT ["migrate"]
