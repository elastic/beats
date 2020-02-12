ARG COUCHBASE_VERSION
FROM couchbase:${COUCHBASE_VERSION}
HEALTHCHECK --interval=1s --retries=90 CMD [ "$(curl -s -o /dev/null -w ''%{http_code}'' http://localhost:8091/pools/default/buckets/beer-sample)" -eq "200" ]
COPY configure-node.sh /opt/couchbase

CMD ["/opt/couchbase/configure-node.sh"]
