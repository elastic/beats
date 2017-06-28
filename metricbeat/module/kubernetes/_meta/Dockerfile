FROM exekias/localkube-image
RUN apt-get update && apt-get install -y curl && apt-get clean
HEALTHCHECK CMD curl -f http://localhost:10255/healthz || exit 1
CMD exec /localkube start \
    --apiserver-insecure-address=0.0.0.0 \
    --apiserver-insecure-port=8080 \
    --logtostderr=true \
    --containerized
