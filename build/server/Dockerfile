FROM --platform=$BUILDPLATFORM golang:1.26.5 AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -o /out/docflow-server ./cmd/docflow-server

FROM gcr.io/distroless/static-debian12:nonroot

ARG VERSION=dev
LABEL org.opencontainers.image.title="Docflow Server" \
      org.opencontainers.image.description="Docflow management API and transactional outbox worker" \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.source="https://github.com/Volkov-D-A/docs-register-and-track"

COPY --from=build /out/docflow-server /usr/local/bin/docflow-server
USER nonroot:nonroot
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/usr/local/bin/docflow-server", "healthcheck"]
ENTRYPOINT ["/usr/local/bin/docflow-server"]
CMD ["run"]
