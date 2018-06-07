# Tomcat is started to fetch Jolokia metrics from it
FROM java:8-jdk-alpine

ENV TOMCAT_VERSION 7.0.86
ENV TC apache-tomcat-${TOMCAT_VERSION}
ENV JOLOKIA_VERSION 1.5.0

RUN apk update;\
    apk add curl

HEALTHCHECK --interval=1s --retries=90 CMD curl -f localhost:8778/jolokia/
EXPOSE 8778

# Prepare a server where jolokia runs in proxy mode
RUN wget http://archive.apache.org/dist/tomcat/tomcat-7/v${TOMCAT_VERSION}/bin/${TC}.tar.gz;\
    tar xzf ${TC}.tar.gz -C /usr;\
    rm ${TC}.tar.gz;\
    sed -i -e 's/Connector port="8080"/Connector port="8778"/g' /usr/${TC}/conf/server.xml;\
    wget http://central.maven.org/maven2/org/jolokia/jolokia-war/${JOLOKIA_VERSION}/jolokia-war-${JOLOKIA_VERSION}.war -O /usr/${TC}/webapps/jolokia.war

# JMX setting to request authentication with remote connection
RUN echo "monitorRole QED" >> /usr/lib/jvm/java-1.8-openjdk/jre/lib/management/jmxremote.password;\
    echo "controlRole R&D" >> /usr/lib/jvm/java-1.8-openjdk/jre/lib/management/jmxremote.password;\
    chmod 600 /usr/lib/jvm/java-1.8-openjdk/jre/lib/management/jmxremote.password

ADD jolokia.xml /usr/${TC}/conf/Catalina/localhost/jolokia.xml

# Start tomcat to accept JMX connection and enable jolokia proxy mode
CMD env CATALINA_OPTS="\
    -Dcom.sun.management.jmxremote.port=7091\
    -Dcom.sun.management.jmxremote.ssl=false\
    -Dcom.sun.management.jmxremote.authenticate=true\
    -Dorg.jolokia.jsr160ProxyEnabled=true" /usr/${TC}/bin/catalina.sh run
