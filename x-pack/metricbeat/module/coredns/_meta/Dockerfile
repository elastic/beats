# Start from coredns base Docker image
FROM coredns/coredns:1.5.0

# create an enhanced container with nc command available since coredns is based
# on scratch image making healthcheck impossible
FROM alpine:latest
COPY --from=0 /coredns /coredns
# Expose client, management, and cluster ports
# For DNS
EXPOSE 53 53/udp

# For Prometheus metrics
EXPOSE 9153 9153/tcp

# Copy coredns configuration in container
ADD config /etc/coredns

RUN apk add --update --no-cache bind-tools

# Check if the Coredns container is healthy
HEALTHCHECK --interval=5s --retries=10 CMD dig @0.0.0.0 my.domain.elastic +dnssec >/dev/null

# Start coredns with custom configuration file
ENTRYPOINT ["/coredns"]
CMD ["-conf", "/etc/coredns/Corefile"]
