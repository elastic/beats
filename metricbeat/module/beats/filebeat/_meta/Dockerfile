FROM debian

ENV VERSION 6.0.0-alpha1-SNAPSHOT
ENV URL https://snapshots.elastic.co/downloads/beats/filebeat/filebeat-${VERSION}-amd64.deb

RUN apt-get update && apt-get install -y wget
RUN wget ${URL}
RUN dpkg -i filebeat-${VERSION}-amd64.deb

COPY filebeat.yml /etc/filebeat/filebeat.yml

EXPOSE 6060

ENTRYPOINT ["filebeat.sh"]

CMD ["-httpprof", ":6060"]
