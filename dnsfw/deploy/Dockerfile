FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    cd cmd/dnsfw && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/dnsfw

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/dnsfw /dnsfw
EXPOSE 53/udp 53/tcp 8080
USER nonroot:nonroot
ENTRYPOINT ["/dnsfw"]
CMD ["-config","/config/config.yaml"]


