FROM ceph/demo:tag-build-master-jewel-centos-7

ENV MON_IP 0.0.0.0
ENV CEPH_PUBLIC_NETWORK 0.0.0.0/0

HEALTHCHECK --interval=1s --retries=90 CMD ceph --status | grep HEALTH_OK
EXPOSE 5000
