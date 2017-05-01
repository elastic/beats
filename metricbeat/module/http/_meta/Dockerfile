# Tomcat is started to fetch Jolokia metrics from it
FROM jolokia/java-jolokia:7
ENV TOMCAT_VERSION 7.0.55
ENV TC apache-tomcat-${TOMCAT_VERSION}

HEALTHCHECK CMD curl -f curl localhost:8778/jolokia/
EXPOSE 8778
RUN wget http://archive.apache.org/dist/tomcat/tomcat-7/v${TOMCAT_VERSION}/bin/${TC}.tar.gz
RUN tar xzf ${TC}.tar.gz -C /opt

CMD env CATALINA_OPTS=$(jolokia_opts) /opt/${TC}/bin/catalina.sh run
