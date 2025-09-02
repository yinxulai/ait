FROM --platform=linux/x86_64 alpine:3.19.1

# 指定工作目录
WORKDIR /workspace

# 安装基本运行环境
RUN apk add bash make util-linux go openssh-client git

# 设置 go 的全局 bin 变量
ENV PATH=$PATH:/root/go/bin

# 安装 go 开发需要的套件
RUN go install github.com/cosmtrek/air@v1.49
