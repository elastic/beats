

FROM centos:7

LABEL \
  org.label-schema.schema-version="1.0" \
  org.label-schema.vendor="Elastic" \
  org.label-schema.name="agent" \
  org.label-schema.version="8.0.0" \
  org.label-schema.url="https://www.elastic.co/products/beats/agent" \
  org.label-schema.vcs-url="github.com/elastic/fleet" \
  org.label-schema.vcs-ref="e108be7c41b66bbf308f610d969a329ee6865fe7" \
  license="Elastic License" \
  description=""

ENV ELASTIC_CONTAINER "true"
ENV PATH=/usr/share/agent:$PATH

COPY beat /usr/share/agent
COPY docker-entrypoint /usr/local/bin/docker-entrypoint
RUN chmod 755 /usr/local/bin/docker-entrypoint

RUN groupadd --gid 1000 agent

RUN mkdir /usr/share/agent/data /usr/share/agent/logs && \
    chown -R root:agent /usr/share/agent && \
    find /usr/share/agent -type d -exec chmod 0750 {} \; && \
    find /usr/share/agent -type f -exec chmod 0640 {} \; && \
    chmod 0750 /usr/share/agent/agent && \
    chmod 0770 /usr/share/agent/data /usr/share/agent/logs
RUN useradd -M --uid 1000 --gid 1000 --home /usr/share/agent agent
USER agent

WORKDIR /usr/share/agent
ENTRYPOINT ["/usr/local/bin/docker-entrypoint"]
CMD ["-e"]
