FROM ibmcom/mq:9.1.3.0

HEALTHCHECK --interval=1s --retries=90 CMD curl -f --insecure https://0.0.0.0:9443/ibmmq/console/login.html

ENV LICENSE=accept
ENV MQ_QMGR_NAME=QM1
ENV MQ_APP_PASSWORD=passw0rd
EXPOSE 1414
EXPOSE 9443
