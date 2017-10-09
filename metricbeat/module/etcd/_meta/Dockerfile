FROM quay.io/coreos/etcd:v3.2
HEALTHCHECK --interval=1s --retries=90 CMD wget -O - http://localhost:2379/health | grep true
