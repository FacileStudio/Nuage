FROM golang:1.24-alpine AS build

ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /repo/apps/api

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api ./

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH:-amd64} go build -o bin/api .

FROM alpine:3.20

RUN apk add --no-cache wget

WORKDIR /app

COPY --from=build /repo/apps/api/bin/api /app/api

EXPOSE 4000

ENTRYPOINT ["/app/api"]
