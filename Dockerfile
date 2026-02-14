FROM zot.ooxo.cc/alpine:latest as certs
RUN apk --update add ca-certificates

FROM zot.ooxo.cc/distroless/static:latest

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app
COPY bin/my_app ./my_app

CMD ["/app/my_app"]
