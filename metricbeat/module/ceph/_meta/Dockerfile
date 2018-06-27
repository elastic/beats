FROM ceph/demo:tag-build-master-jewel-centos-7

ENV MON_IP 0.0.0.0
ENV CEPH_PUBLIC_NETWORK 0.0.0.0/0

RUN yum -q install -y jq && yum clean all && rm -fr /var/cache/yum

# Wait for the health endpoint to have monitors information
HEALTHCHECK --interval=1s --retries=300 \
  CMD curl -s -H "Accept: application/json" localhost:5000/api/v0.1/health \
        | jq .output.health.health_services[0].mons[0] \
        | grep health
EXPOSE 5000
