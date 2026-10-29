# syntax=docker/dockerfile:1
# SPDX-License-Identifier: AGPL-3.0-only
#
# Tuskline server image (roadmap M0.9): web build -> Go build -> distroless.
# Runs `tuskline serve` on :8080 as a non-root user.
#
#   docker build -t tuskline .
#   docker run --rm -p 8080:8080 tuskline
#
# Base image versions follow .prototools (Node.js, Go); Dependabot bumps them.

# --- web: SvelteKit static build -------------------------------------------
FROM node:26.13.0-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# --- go: static tuskline binary with the UI embedded ------------------------
FROM golang:1.27.3-alpine AS go
ENV CGO_ENABLED=0 GOFLAGS=-trimpath GOTOOLCHAIN=local
WORKDIR /src
# Every module named in go.work must exist; cli and agent only need go.mod.
COPY go.work ./
COPY core/go.mod core/go.sum core/
COPY server/go.mod server/
COPY cli/go.mod cli/go.sum cli/
COPY agent/go.mod agent/
RUN cd server && go mod download
COPY core/ core/
COPY server/ server/
COPY --from=web /src/web/build/ server/internal/webui/dist/
ARG VERSION=0.0.0-dev
ARG COMMIT=none
ARG DATE=unknown
RUN cd server && go build \
	-ldflags "-s -w \
	-X github.com/interlace-sh/tuskline/core/version.Version=${VERSION} \
	-X github.com/interlace-sh/tuskline/core/version.Commit=${COMMIT} \
	-X github.com/interlace-sh/tuskline/core/version.Date=${DATE}" \
	-o /out/tuskline ./cmd/tuskline
# The future SQLite store lives in /data; create it owned by the nonroot user.
RUN mkdir -p /out/data

# --- runtime: distroless, no shell, non-root ---------------------------------
FROM gcr.io/distroless/static-debian13:nonroot
LABEL org.opencontainers.image.title="tuskline" \
	org.opencontainers.image.description="Tuskline server: PostgreSQL workbench and control plane" \
	org.opencontainers.image.source="https://github.com/interlace-sh/tuskline" \
	org.opencontainers.image.licenses="AGPL-3.0-only AND Apache-2.0"
COPY --from=go /out/tuskline /usr/local/bin/tuskline
COPY --from=go --chown=nonroot:nonroot /out/data /data
COPY LICENSE LICENSING.md /usr/share/doc/tuskline/
COPY core/LICENSE /usr/share/doc/tuskline/LICENSE.core
# The binary defaults to 127.0.0.1:8080; inside a container it must listen on
# every interface so `-p 8080:8080` works.
ENV TUSKLINE_ADDR=:8080
EXPOSE 8080
VOLUME ["/data"]
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/tuskline"]
CMD ["serve"]
