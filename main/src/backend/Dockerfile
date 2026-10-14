# syntax=docker/dockerfile:1

# RYZE API image.
#
# Built from the backend module directory, so go.mod/go.sum are at the context
# root:
#
#   docker build -t ryze-api ./main/src/backend
#
# The released version is injected at build time so GET /api/v1/version reports
# the same value that is recorded in the release notes.

FROM golang:1.27 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags "-s -w -X ryze/backend/version.Version=${VERSION}" \
      -o /out/ryze-api .

# The runtime image is distroless: no shell, no package manager and a non-root
# user by default. It ships the CA certificates the API needs to call Stripe and
# PayPal over HTTPS.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/ryze-api /ryze-api

EXPOSE 8080

USER nonroot:nonroot

# Migrations are run explicitly against the deployed database, never on boot:
#   docker run --rm --env-file .env ryze-api migrate up
ENTRYPOINT ["/ryze-api"]
