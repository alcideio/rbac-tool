FROM alpine:latest
RUN  apk --no-cache --update add bash wget ca-certificates

WORKDIR /
COPY rbac-tool /rbac-tool

ENTRYPOINT  ["/rbac-tool"]