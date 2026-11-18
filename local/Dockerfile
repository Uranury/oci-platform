# Local development image: builds the same single binary that production
# runs (migrations, seed data and the frontend are all embedded), on a
# minimal non-root base. Production deploys go through ansible/, not this.
#
# Build from the repo root:  docker build -f local/Dockerfile .
# Usually run via local/compose.yaml.

ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
# The app binds 127.0.0.1 by default; inside a container it must listen on
# all interfaces for the published port to reach it.
ENV LISTEN_HOST=0.0.0.0 PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app"]
