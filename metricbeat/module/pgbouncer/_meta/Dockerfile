FROM python:3.9.18-bullseye as build

COPY . beats
RUN \
    --mount=type=cache,target=/var/cache/go \
    wget https://dl.google.com/go/go1.20.10.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.20.10.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin
RUN mkdir -p /root/go/bin
RUN git clone https://github.com/magefile/mage
RUN cd mage; go run bootstrap.go
RUN cp /root/go/bin/mage /usr/local/go/bin/

WORKDIR /beats/metricbeat
RUN \
    --mount=type=cache,target=/var/cache/mage \
    mage update && mage build

FROM debian:trixie-slim AS final

RUN mkdir -p /etc/metricbeat
COPY --from=build /beats/metricbeat/metricbeat /usr/bin/metricbeat
COPY --from=build /beats/metricbeat/module/pgbouncer/_meta/config.yml /etc/metricbeat/metricbeat.yml

ENTRYPOINT ["metricbeat", "run", "--path.config", "/etc/metricbeat"]