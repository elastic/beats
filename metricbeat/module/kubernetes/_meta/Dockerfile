FROM exekias/localkube-image
RUN apt-get update && apt-get install -y curl && apt-get clean
HEALTHCHECK --interval=1s --retries=300 CMD curl -f localhost:10255/stats/summary | grep kube-addon-manager
CMD exec /localkube start \
    --apiserver-insecure-address=0.0.0.0 \
    --apiserver-insecure-port=8080 \
    --logtostderr=true \
    --containerized
