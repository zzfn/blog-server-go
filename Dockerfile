# 使用官方的 Golang 镜像作为构建环境。
FROM golang:1.17 as builder

# 设置工作目录。
WORKDIR /app

# 将代码和依赖项复制到容器中。
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

# 构建应用。
RUN go build -o my_app

# 使用 scratch 作为基础镜像来创建一个最小化的新镜像。
FROM scratch

# 将工作目录设置为 /。
WORKDIR /

# 从构建器镜像中复制二进制文件和其他必要的文件。
COPY --from=builder /app/my_app /my_app

# 运行应用。
CMD ["/my_app"]