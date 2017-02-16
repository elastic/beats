# Copy of https://github.com/elastic/kibana-docker/blob/master/build/kibana/Dockerfile
FROM docker.elastic.co/kibana/kibana-ubuntu-base:latest
MAINTAINER Elastic Docker Team <docker@elastic.co>

ARG DOWNLOAD_URL
ARG ELASTIC_VERSION

EXPOSE 5601

WORKDIR /usr/share/kibana
RUN curl -Ls ${DOWNLOAD_URL}/kibana/kibana-${ELASTIC_VERSION}-linux-x86_64.tar.gz | tar --strip-components=1 -zxf - && \
    #bin/kibana-plugin install ${DOWNLOAD_URL}/kibana-plugins/x-pack/x-pack-${ELASTIC_VERSION}.zip} && \
    ln -s /usr/share/kibana /opt/kibana

# Set some Kibana configuration defaults.
ADD config/kibana.yml /usr/share/kibana/config/

# Add the launcher/wrapper script. It knows how to interpret environment
# variables and translate them to Kibana CLI options.
ADD bin/kibana-docker /usr/local/bin/

# Add a self-signed SSL certificate for use in examples.
#ADD ssl/kibana.example.org.* /usr/share/kibana/config/

RUN usermod --home /usr/share/kibana kibana
USER kibana
ENV PATH=/usr/share/kibana/bin:$PATH
CMD /usr/local/bin/kibana-docker
