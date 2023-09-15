FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app
COPY bin/my_app ./my_app

CMD ["/app/my_app"]
