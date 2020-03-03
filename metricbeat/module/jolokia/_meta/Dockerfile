# Tomcat is started to fetch Jolokia metrics from it
FROM java:8-jdk-alpine

ENV TOMCAT_VERSION 7.0.86
ENV TC apache-tomcat-${TOMCAT_VERSION}
ARG JOLOKIA_VERSION

RUN apk update && \
    apk add curl openssl ca-certificates bash

HEALTHCHECK --interval=1s --retries=90 CMD curl -f localhost:8778/jolokia/
EXPOSE 8778

COPY jolokia-${JOLOKIA_VERSION}.sum jolokia.sum

# Prepare a server where jolokia runs in proxy mode
RUN wget http://archive.apache.org/dist/tomcat/tomcat-7/v${TOMCAT_VERSION}/bin/${TC}.tar.gz && \
    tar xzf ${TC}.tar.gz -C /usr && \
    rm ${TC}.tar.gz && \
    sed -i -e 's/Connector port="8080"/Connector port="8778"/g' /usr/${TC}/conf/server.xml
RUN curl -J -L -s -f -o - https://github.com/kadwanev/retry/releases/download/1.0.1/retry-1.0.1.tar.gz | tar xfz - -C /usr/local/bin
RUN retry --min 1 --max 180 -- curl -J -L -s -f --show-error -O \
        "https://repo1.maven.org/maven2/org/jolokia/jolokia-war/${JOLOKIA_VERSION}/jolokia-war-${JOLOKIA_VERSION}.war" && \
    sha256sum -c jolokia.sum && \
    mv jolokia-war-${JOLOKIA_VERSION}.war /usr/${TC}/webapps/jolokia.war && rm jolokia.sum

# JMX setting to request authentication with remote connection
RUN echo "monitorRole QED" >> /usr/lib/jvm/java-1.8-openjdk/jre/lib/management/jmxremote.password && \
    echo "controlRole R&D" >> /usr/lib/jvm/java-1.8-openjdk/jre/lib/management/jmxremote.password && \
    chmod 600 /usr/lib/jvm/java-1.8-openjdk/jre/lib/management/jmxremote.password

ADD jolokia.xml /usr/${TC}/conf/Catalina/localhost/jolokia.xml

# Start tomcat to accept JMX connection and enable jolokia proxy mode
CMD env CATALINA_OPTS="\
    -Dcom.sun.management.jmxremote.port=7091\
    -Dcom.sun.management.jmxremote.ssl=false\
    -Dcom.sun.management.jmxremote.authenticate=true\
    -Dorg.jolokia.jsr160ProxyEnabled=true" /usr/${TC}/bin/catalina.sh run
