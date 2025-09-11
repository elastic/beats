ARG COUCHDB_VERSION
FROM apache/couchdb:${COUCHDB_VERSION}

HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost:5984/ | grep Welcome

EXPOSE 5984
