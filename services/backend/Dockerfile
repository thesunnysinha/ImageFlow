FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
RUN mkdir -p /out/data/images

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/server /server
# Writable directory for processed images (mount a volume here); owned by the numeric runtime user.
COPY --from=build --chown=10001:10001 /out/data /data
# Numeric UID so Kubernetes can enforce runAsNonRoot.
USER 10001
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD ["/server", "-healthcheck"]
ENTRYPOINT ["/server"]
