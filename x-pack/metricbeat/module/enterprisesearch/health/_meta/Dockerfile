ARG ENT_VERSION
FROM docker.elastic.co//enterprise-search/enterprise-search:${ENT_VERSION}

COPY docker-entrypoint-dependencies.sh /usr/local/bin/
ENTRYPOINT /usr/local/bin/docker-entrypoint-dependencies.sh

HEALTHCHECK --interval=1s --retries=300 --start-period=60s CMD python -c 'import urllib, json; response = urllib.urlopen("http://elastic:changeme@localhost:3002/api/as/v1/internal/health"); response.getcode() == 200 or exit(1); data = json.loads(response.read()); exit(0);'
