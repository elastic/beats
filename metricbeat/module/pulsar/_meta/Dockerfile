ARG PULSAR_VERSION
FROM apachepulsar/pulsar:$PULSAR_VERSION
CMD ["bin/pulsar", "standalone"]
EXPOSE 8080
HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost:8080/admin/v2/brokers/health
