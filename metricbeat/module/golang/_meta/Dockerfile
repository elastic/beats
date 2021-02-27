FROM docker.elastic.co/beats/metricbeat:6.5.4

HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost:6060/debug/vars

EXPOSE 6060
CMD ["-httpprof", ":6060", "-e"]
