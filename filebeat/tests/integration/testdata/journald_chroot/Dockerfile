FROM alpine:latest AS prep

FROM scratch

COPY --from=prep /etc/alpine-release /etc/alpine-release
COPY filebeat /
COPY filebeat.yml /
ENTRYPOINT ["/filebeat"]
CMD ["-c", "/filebeat.yml", "--strict.perms=false"]
