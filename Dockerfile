FROM alpine:latest AS build
RUN  apk --no-cache --update add wget ca-certificates

FROM scratch

WORKDIR /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY rbac-tool /rbac-tool

ENTRYPOINT  ["/rbac-tool"]
