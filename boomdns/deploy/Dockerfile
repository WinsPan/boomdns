FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    cd cmd/boomdns && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/boomdns

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/boomdns /boomdns
EXPOSE 53/udp 53/tcp 8080
USER nonroot:nonroot
ENTRYPOINT ["/boomdns"]
CMD ["-config","/config/config.yaml"]


