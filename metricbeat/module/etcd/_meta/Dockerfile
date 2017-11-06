FROM quay.io/coreos/etcd:v3.2
HEALTHCHECK --interval=1s --retries=90 CMD wget -O - http://localhost:2379/health | grep true
CMD ["/usr/local/bin/etcd", "--advertise-client-urls", "http://0.0.0.0:2379,http://0.0.0.0:4001", "--listen-client-urls", "http://0.0.0.0:2379,http://0.0.0.0:4001"]
