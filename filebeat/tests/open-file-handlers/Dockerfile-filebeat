FROM debian

RUN apt-get update && apt-get install -y wget
RUN wget https://beats-nightlies.s3.amazonaws.com/filebeat/filebeat-6.0.0-alpha1-SNAPSHOT-linux-x86_64.tar.gz -O filebeat.tar.gz
RUN mkdir filebeat
RUN tar xvfz filebeat.tar.gz -C filebeat --strip-components=1
RUN wget https://beats-nightlies.s3.amazonaws.com/metricbeat/metricbeat-6.0.0-alpha1-SNAPSHOT-amd64.deb -O metricbeat.deb
RUN dpkg --force-overwrite -i metricbeat.deb

COPY filebeat.yml /filebeat/filebeat.yml
COPY metricbeat.yml /etc/metricbeat/metricbeat.yml
COPY run.sh /run.sh

ENTRYPOINT ["sh", "run.sh"]
