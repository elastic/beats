FROM java:8-jre

ENV ELASTICSEARCH_MAJOR 5.0
ENV ELASTICSEARCH_VERSION 5.0
ENV VERSION 5.0.0
ENV FILENAME_VERSION 5.0.0
ENV URL http://staging.elastic.co/5.0.0-ccd69424/downloads/elasticsearch/elasticsearch-5.0.0.tar.gz

ENV ESHOME /opt/elasticsearch-${FILENAME_VERSION}

# grab gosu for easy step-down from root
RUN gpg --keyserver ha.pool.sks-keyservers.net --recv-keys B42F6819007F00F88E364FD4036A9C25BF357DD4
RUN arch="$(dpkg --print-architecture)" \
	&& set -x \
	&& curl -o /usr/local/bin/gosu -fSL "https://github.com/tianon/gosu/releases/download/1.3/gosu-$arch" \
	&& curl -o /usr/local/bin/gosu.asc -fSL "https://github.com/tianon/gosu/releases/download/1.3/gosu-$arch.asc" \
	&& gpg --verify /usr/local/bin/gosu.asc \
	&& rm /usr/local/bin/gosu.asc \
	&& chmod +x /usr/local/bin/gosu

RUN groupadd -r elasticsearch && useradd -r -m -g elasticsearch elasticsearch

RUN set -x && \
	cd /opt && \
	wget -qO elasticsearch.tar.gz "$URL?t=$(date +%F)" && \
	tar xzvf elasticsearch.tar.gz && \
	chown -R elasticsearch:elasticsearch ${ESHOME}

ENV PATH ${ESHOME}/bin:$PATH

VOLUME ${ESHOME}/data

ENV ES_JAVA_OPTS="-Xms512m -Xmx512m"

# Shield currently not enabled
#ES_JAVA_OPTS="-Des.plugins.staging=b0da471" elasticsearch-plugin install x-pack

COPY docker-entrypoint.sh /

ENTRYPOINT ["/docker-entrypoint.sh"]

EXPOSE 9200 9300

CMD ["elasticsearch"]

