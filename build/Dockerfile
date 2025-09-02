# 第一阶段：构建二进制文件
FROM golang:1.20 AS builder

WORKDIR /workspace

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go

# 第二阶段：构建镜像
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

EXPOSE 8080

COPY --from=builder /workspace/main .

CMD ["./main"]
