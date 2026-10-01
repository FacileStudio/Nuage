FROM oven/bun:1.3.13 AS client-build
WORKDIR /client
COPY apps/client/package.json apps/client/bun.lock* ./
RUN bun install --frozen-lockfile
COPY apps/client/ .
RUN bun run build

FROM golang:1.26.8-alpine AS api-build

ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /repo/apps/api

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api ./

# The commit is read from the repository the context was built out of and linked
# in, so a running container can name the revision it is serving. That is why
# `.git` is the one thing .dockerignore keeps. -buildvcs=false because the
# stamp is the answer here, not the toolchain's own guess at a partial checkout.
COPY .git /repo/.git
RUN apk add --no-cache git

RUN commit="$(git -C /repo rev-parse --short=12 HEAD 2>/dev/null)"; \
    ldflags="-s -w"; \
    [ -z "$commit" ] || ldflags="$ldflags -X github.com/FacileStudio/Nuage/apps/api/internal/version.stamp=$commit"; \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -buildvcs=false -ldflags="$ldflags" -o bin/api .

# STORAGE_DIR, which only the API writes to, and only for avatars. distroless
# has no shell, so it is made here and copied into the final image owned by the
# user the container runs as: root-owned, the boot-time MkdirAll for avatars
# fails and the API exits instead of serving.
RUN mkdir -p /repo/app/data/avatars

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=api-build /repo/apps/api/bin/api /api
COPY --from=client-build /client/build /client
COPY --from=api-build --chown=nonroot:nonroot /repo/app/data /app/data

# The distroless base can carry its own WorkingDir (/home/nonroot on the
# :nonroot variant), which would make a relative ./client resolve there and
# the SPA silently not be served at all. Be explicit.
ENV CLIENT_DIR=/client

EXPOSE 4000

# The base is already :nonroot; saying so again keeps the intent if the base
# ever moves. Nothing here needs root, and nobody should be able to hand it to
# the process either.
USER nonroot:nonroot

ENTRYPOINT ["/api"]
