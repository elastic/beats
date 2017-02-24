FROM couchbase:4.5.1
HEALTHCHECK CMD curl -f http://localhost:8091

COPY configure-node.sh /opt/couchbase

CMD ["/opt/couchbase/configure-node.sh"]
