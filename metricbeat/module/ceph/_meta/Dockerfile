FROM ceph/daemon:master-6373c6a-jewel-centos-7-x86_64

HEALTHCHECK --interval=1s --retries=90 CMD ceph --status | grep HEALTH_OK
EXPOSE 5000

ENV NETWORK_AUTO_DETECT 4
ENV DEMO_DAEMONS osd,rest_api

CMD ["demo"]
