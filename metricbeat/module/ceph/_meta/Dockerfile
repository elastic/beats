FROM ceph/demo:tag-build-master-jewel-centos-7

ENV MON_IP 0.0.0.0
ENV CEPH_PUBLIC_NETWORK 0.0.0.0/0

RUN yum install -y nc && yum clean all
HEALTHCHECK CMD nc -w 1 -v 127.0.0.1 5000 </dev/null
EXPOSE 5000
