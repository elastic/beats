ARG TAG=1.7
FROM apache/couchdb:${TAG}
COPY ./local.ini /etc/couchdb/local.ini
EXPOSE 5984
HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost:5984/ | grep Welcome
