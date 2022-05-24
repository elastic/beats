FROM golang:1.17.8 as builder

ENV PATH=/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin:/go/bin:/usr/local/go/bin

RUN go install github.com/go-delve/delve/cmd/dlv@latest

WORKDIR /usr/share/beats
RUN mkdir /usr/share/beats/build

COPY . /usr/share/beats

RUN go build -gcflags "-N -l" -o /usr/share/beats/build/metricbeat metricbeat/main.go



FROM golang:1.17.8

ENV PATH=/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin:/go/bin:/usr/local/go/bin

WORKDIR /usr/share/beats

ENV ELASTICSEARCH_PASSWORD=changeme
ENV ELASTICSEARCH_USERNAME=elastic
ENV ELASTICSEARCH_HOST=elasticsearch

COPY --from=builder /usr/share/beats/build/metricbeat /usr/share/beats/metricbeat
COPY --from=builder /go/bin/dlv /go/bin/dlv
COPY --from=builder /usr/share/beats/metricbeat.yml /usr/share/beats/metricbeat.yml

CMD ["dlv", "--headless=true", "--listen=:56268", "--api-version=2", "--log", "exec", "./metricbeat", "--", "-c", "./metricbeat.yml", "-e"]