FROM ubuntu:14.04
MAINTAINER Monica Sarbu <monica@elastic.co>

ENV METRICBEAT_FILE=metricbeat-6.0.0-alpha1-SNAPSHOT-linux-x86_64

# Cache variable can be set during building to invalidate the build cache with `--build-arg CACHE=$(date +%s) .`
ARG CACHE=1

ADD https://beats-nightlies.s3.amazonaws.com/metricbeat/$METRICBEAT_FILE.tar.gz?${CACHE} /$METRICBEAT_FILE.tar.gz

RUN tar -xzvf $METRICBEAT_FILE.tar.gz && \
    ln -s $METRICBEAT_FILE metricbeat

EXPOSE 8080
ENTRYPOINT ["/metricbeat/metricbeat", "-httpprof", "0.0.0.0:8080", "-c", "/metricbeat/metricbeat.yml", "-e", "-v"]
