ARG ENT_VERSION
FROM docker.elastic.co/enterprise-search/enterprise-search:${ENT_VERSION}

COPY docker-entrypoint-dependencies.sh /usr/local/bin/
# We need to explicitly specify tini here or Docker will use /bin/sh to run the script and
# on Debian-based images (which we use for ARM64 images) it runs dash, which does not
# support environment variables with dots and it leads to all config options being dropped
# See https://github.com/docker-library/openjdk/issues/135#issuecomment-318495067
ENTRYPOINT ["tini", "--", "/usr/local/bin/docker-entrypoint-dependencies.sh"]

HEALTHCHECK --interval=1s --retries=300 --start-period=60s \
  CMD curl --user elastic:changeme --fail --silent http://localhost:3002/api/ent/v1/internal/health
