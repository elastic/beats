FROM apache/couchdb:1.7
COPY ./local.ini /etc/couchdb/local.ini
EXPOSE 5984 
HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost:5984/ | grep Welcome
