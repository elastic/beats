# Copy of https://github.com/elastic/kibana-docker/blob/master/build/kibana/Dockerfile
FROM centos:7
LABEL maintainer "Elastic Docker Team <docker@elastic.co>"
EXPOSE 5601

### Beats specific args ####
ARG DOWNLOAD_URL
ARG ELASTIC_VERSION
ARG CACHE_BUST=1
ARG IMAGE_FLAVOR=x-pack

# Healthcheck create by beats team
RUN yum install update -y epel-release && yum install -y jq
HEALTHCHECK --interval=1s --retries=600 CMD curl -f http://localhost:5601/api/status | jq '. | .status.overall.state' | grep -q green
### Beats ###

# Add Reporting dependencies + healthcheck tool
RUN yum update -y && yum install -y fontconfig freetype && yum clean all

WORKDIR /usr/share/kibana
COPY download.sh /download.sh
RUN /download.sh $DOWNLOAD_URL $ELASTIC_VERSION $CACHE_BUST && rm /download.sh
RUN tar --strip-components=1 -zxf kibana-${ELASTIC_VERSION}-linux-x86_64.tar.gz && \
    rm kibana-${ELASTIC_VERSION}-linux-x86_64.tar.gz && \
    ln -s /usr/share/kibana /opt/kibana

ENV ELASTIC_CONTAINER true
ENV PATH=/usr/share/kibana/bin:$PATH

# Set some Kibana configuration defaults.
COPY config/kibana-x-pack.yml /usr/share/kibana/config/kibana.yml

# Add the launcher/wrapper script. It knows how to interpret environment
# variables and translate them to Kibana CLI options.
COPY bin/kibana-docker /usr/local/bin/

# Provide a non-root user to run the process.
RUN groupadd --gid 1000 kibana && \
    useradd --uid 1000 --gid 1000 \
      --home-dir /usr/share/kibana --no-create-home \
      kibana
USER kibana

CMD ["/bin/bash", "/usr/local/bin/kibana-docker"]

