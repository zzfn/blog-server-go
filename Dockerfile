FROM scratch

WORKDIR /app

# 从构建器镜像中复制二进制文件和其他必要的文件。
COPY /app/bin/my_app /my_app

# 运行应用。
CMD ["/my_app"]