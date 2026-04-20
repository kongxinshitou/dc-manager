FROM golang:latest AS builder

WORKDIR /app

# 先复制依赖文件，利用 Docker 层缓存
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

COPY . .

# 纯 Go 编译，无 CGO 依赖
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o dcmanager .

# ---

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/dcmanager .
# 将 DB 作为初始数据备份，首次启动时由 entrypoint 复制到 volume
COPY dc_manager.db /app/dc_manager.db.init
# Copy Excel layout file for auto-import
COPY 001_中联重科数据中心台账_20260226.xlsx /app/001_中联重科数据中心台账_20260226.xlsx

COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

ENV TZ=Asia/Shanghai
ENV PORT=8080
ENV DB_PATH=/app/data/dc_manager.db

EXPOSE 8080

ENTRYPOINT ["/app/entrypoint.sh"]
