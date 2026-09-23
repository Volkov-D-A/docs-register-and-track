FROM --platform=$BUILDPLATFORM golang:1.27.1 AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG BUILD_IDENTITY
ARG LOCAL_BUILD=0
ARG COVERAGE=0
ARG VERSION

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN test "$(go run ./cmd/docflow-server version)" = "$VERSION" \
    && flags="$(DOCFLOW_LOCAL_BUILD=$LOCAL_BUILD go run ./tools/buildmeta from-identity "$BUILD_IDENTITY")" \
    && if [ "$COVERAGE" = "1" ]; then \
         CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -cover -trimpath -ldflags "$flags" -o /out/docflow-server ./cmd/docflow-server; \
       else \
         CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags "$flags" -o /out/docflow-server ./cmd/docflow-server; \
       fi

FROM postgres:18.3-alpine

ARG VERSION=dev
ARG BUILD_IDENTITY
LABEL org.opencontainers.image.title="Docflow Server" \
      org.opencontainers.image.description="Docflow management API and transactional outbox worker" \
      org.opencontainers.image.version=$VERSION \
      io.docflow.build.identity=$BUILD_IDENTITY \
      org.opencontainers.image.source="https://github.com/Volkov-D-A/docs-register-and-track"

COPY --from=build /out/docflow-server /usr/local/bin/docflow-server
RUN mkdir -p /var/lib/docflow/backups && chown -R 65532:65532 /var/lib/docflow
USER 65532:65532
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/usr/local/bin/docflow-server", "healthcheck"]
ENTRYPOINT ["/usr/local/bin/docflow-server"]
CMD ["run"]
