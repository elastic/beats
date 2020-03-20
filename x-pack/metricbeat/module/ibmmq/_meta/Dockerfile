ARG IBMMQ_VERSION

FROM ibmcom/mq:${IBMMQ_VERSION}

ENV IBMMQ_METRICS_REST_PORT=9157

ENV LICENSE=accept
ENV MQ_QMGR_NAME=QM1
ENV MQ_ENABLE_METRICS=true

HEALTHCHECK --interval=1s --retries=90 CMD curl -s --fail http://127.0.0.1:${IBMMQ_METRICS_REST_PORT}/metrics | grep -q "ibmmq_qmgr_commit_total"
