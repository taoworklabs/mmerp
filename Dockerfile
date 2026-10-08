FROM node:24-slim AS web
RUN npm install -g pnpm@11
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.27 AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -tags embedweb \
    -ldflags "-s -w -X github.com/taoworklabs/mmerp/internal/app.Version=${VERSION}" \
    -o /out/mmerp ./cmd/server

FROM debian:trixie-slim
COPY --from=server /out/mmerp /usr/local/bin/mmerp
# A new named volume copies this directory's owner, so the app can write to it.
RUN mkdir -p /data/files && chown 10001 /data/files
USER 10001
EXPOSE 8080
ENTRYPOINT ["mmerp"]
