FROM alpine:3.21 as certs
RUN apk --update add ca-certificates

FROM scratch

ARG USER_UID=10001
USER ${USER_UID}

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chmod=755 _build/beats-otel-collector /beats-otel-collector
COPY x-pack/otel/distribution/config.yaml /etc/beats-otel-collector/config.yaml
ENTRYPOINT ["/beats-otel-collector"]
CMD ["--config", "/etc/beats-otel-collector/config.yaml"]
EXPOSE 4317 4318 13133 1777 55679