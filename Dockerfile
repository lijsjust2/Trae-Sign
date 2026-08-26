# 多阶段构建：前端(Vue) → 后端(Go) → 运行镜像(alpine)
# 多架构：buildx 自动注入 TARGETOS/TARGETARCH，支持 linux/amd64 与 linux/arm64

# ===== 阶段 1：构建前端 =====
FROM node:20-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ===== 阶段 2：构建后端 =====
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod ./
COPY main.go ./
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o trae-signin-web .

# ===== 阶段 3：运行镜像 =====
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /src/trae-signin-web /app/trae-signin-web
COPY --from=web /web/dist /app/web/dist

# 数据目录（data.json / auth.json 持久化在这里）
VOLUME /app/data
ENV PORT=9090
ENV TZ=Asia/Shanghai

EXPOSE 9090
ENTRYPOINT ["/app/trae-signin-web"]
