FROM docker.elastic.co/observability/stream:v0.19.0

RUN apk add --no-cache curl

ENV PORT="8080"
COPY files /files
HEALTHCHECK --interval=1s --retries=600 CMD curl -X POST http://localhost:8080/token
CMD [ "http-server", "--addr=:8080", "--config=/files/config.yml" ]
