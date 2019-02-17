# Start from coredns base Docker image
FROM coredns/coredns:latest

# For DNS
EXPOSE 53 53/udp

# For Prometheus metrics
EXPOSE 9153 9153/tcp

# Mount coredns configuration in container
VOLUME ["/etc/coredns"]

# Start coredns with custom configuration file
ENTRYPOINT ["/coredns"]
CMD ["-conf", "/etc/coredns/Corefile"]
