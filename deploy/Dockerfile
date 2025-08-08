ARG GO_IMAGE=golang:1.22-alpine
ARG RUNTIME_IMAGE=gcr.io/distroless/static:nonroot
FROM ${GO_IMAGE} AS build
WORKDIR /src

# 可通过 --build-arg GOPROXY=... 覆盖
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY} \
    GO111MODULE=on

# 构建依赖
RUN apk add --no-cache git ca-certificates

# 预缓存依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建可执行文件，确保 /out 目录存在
RUN mkdir -p /out && \
    cd cmd/boomdns && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/boomdns

FROM ${RUNTIME_IMAGE}
WORKDIR /
COPY --from=build /out/boomdns /boomdns
EXPOSE 53/udp 53/tcp 8080
USER nonroot:nonroot
ENTRYPOINT ["/boomdns"]
CMD ["-config","/config/config.yaml"]


