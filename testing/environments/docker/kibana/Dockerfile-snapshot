# Copy of https://github.com/elastic/kibana-docker/blob/master/build/kibana/Dockerfile
FROM docker.elastic.co/kibana/kibana-ubuntu-base:latest
MAINTAINER Elastic Docker Team <docker@elastic.co>

ARG DOWNLOAD_URL
ARG ELASTIC_VERSION
ARG CACHE_BUST=1
ARG XPACK

RUN apt-get update && apt-get install -y jq && apt-get clean

HEALTHCHECK --retries=6 CMD curl -f http://localhost:5601/api/status | jq '. | .status.overall.state' | grep -q green
EXPOSE 5601

WORKDIR /usr/share/kibana
RUN curl -Ls ${DOWNLOAD_URL}/kibana/kibana-${ELASTIC_VERSION}-linux-x86_64.tar.gz?c=${CACHE_BUST} | tar --strip-components=1 -zxf - && \
    ln -s /usr/share/kibana /opt/kibana

# Install XPACK
RUN if [ ${XPACK} = "1" ]; then bin/kibana-plugin install ${DOWNLOAD_URL}/kibana-plugins/x-pack/x-pack-${ELASTIC_VERSION}.zip?c=${CACHE_BUST}; fi

# Set some Kibana configuration defaults.
ADD config/kibana.yml /usr/share/kibana/config/

# Add the launcher/wrapper script. It knows how to interpret environment
# variables and translate them to Kibana CLI options.
ADD bin/kibana-docker /usr/local/bin/

# Add a self-signed SSL certificate for use in examples.
#ADD ssl/kibana.example.org.* /usr/share/kibana/config/

RUN usermod --home /usr/share/kibana kibana
USER kibana
ENV BABEL_CACHE_PATH /usr/share/kibana/optimize/.babelcache.json
ENV PATH=/usr/share/kibana/bin:$PATH
CMD /usr/local/bin/kibana-docker
