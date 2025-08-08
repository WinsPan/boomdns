# syntax=docker/dockerfile:1.5

FROM golang:1.22-alpine AS build
WORKDIR /src

# 可通过 --build-arg GOPROXY=... 覆盖
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY} \
    GO111MODULE=on

# 构建依赖
RUN apk add --no-cache git ca-certificates

# 预缓存依赖
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# 复制源代码
COPY . .

# 构建可执行文件，确保 /out 目录存在
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p /out && \
    cd cmd/boomdns && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/boomdns

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/boomdns /boomdns
EXPOSE 53/udp 53/tcp 8080
USER nonroot:nonroot
ENTRYPOINT ["/boomdns"]
CMD ["-config","/config/config.yaml"]


