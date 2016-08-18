FROM debian

RUN apt-get update && apt-get install -y wget
RUN wget https://download.elastic.co/beats/filebeat/filebeat-5.0.0-alpha5-amd64.deb
RUN dpkg -i filebeat-5.0.0-alpha5-amd64.deb

COPY filebeat.yml /etc/filebeat/filebeat.yml

EXPOSE 6060

ENTRYPOINT ["filebeat.sh"]

CMD ["-httpprof", ":6060"]
