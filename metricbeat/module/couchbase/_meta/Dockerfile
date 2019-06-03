FROM couchbase:4.5.1
HEALTHCHECK --interval=1s --retries=90 CMD [ "$(curl -s -o /dev/null -w ''%{http_code}'' http://localhost:8091/pools/default/buckets/beer-sample)" -eq "200" ]
COPY configure-node.sh /opt/couchbase

CMD ["/opt/couchbase/configure-node.sh"]
