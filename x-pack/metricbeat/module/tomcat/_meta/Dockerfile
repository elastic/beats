ARG TOMCAT_VERSION=9.0.26
FROM tomcat:${TOMCAT_VERSION}-jdk13-openjdk-oracle

RUN apt-get update && apt-get install -y curl

HEALTHCHECK --interval=1s --retries=90 CMD curl -q http://localhost:8080

CMD ["catalina.sh", "run"]
