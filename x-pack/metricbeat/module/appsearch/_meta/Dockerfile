FROM docker.elastic.co/app-search/app-search:7.4.0

RUN echo 'for I in {1..10}; do /usr/share/app-search/bin/app-search "$@" && break || sleep 5; done' >> /usr/local/bin/docker-entrypoint.sh

HEALTHCHECK --interval=1s --retries=300 --start-period=60s CMD python -c 'import urllib, json; response = urllib.urlopen("http://myelastic:changeme@localhost:3002/swiftype-app-version"); data = json.loads(response.read()); exit(0);'
