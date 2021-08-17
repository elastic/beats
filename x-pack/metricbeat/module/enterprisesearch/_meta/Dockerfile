ARG ENT_VERSION
FROM docker.elastic.co/enterprise-search/enterprise-search:${ENT_VERSION}

COPY docker-entrypoint-dependencies.sh /usr/local/bin/
ENTRYPOINT /usr/local/bin/docker-entrypoint-dependencies.sh

HEALTHCHECK --interval=1s --retries=300 --start-period=60s CMD curl -u elastic:changeme -s http://localhost:3002/api/as/v1/internal/health
