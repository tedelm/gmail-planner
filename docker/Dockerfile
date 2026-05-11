# syntax=docker/dockerfile:1

FROM golang:1.25-bookworm AS build
WORKDIR /src

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gmail-planner ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
# Image includes CA bundle for HTTPS (Google APIs, OpenAI).
COPY --from=build --chown=nonroot:nonroot /out/gmail-planner /usr/local/bin/gmail-planner
WORKDIR /app
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/gmail-planner"]
CMD ["-digest"]
