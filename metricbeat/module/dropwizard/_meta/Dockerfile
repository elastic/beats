FROM maven:3.6-jdk-8

# Variables used in pom.xml
ARG DROPWIZARD_VERSION

# Build the deployable war
COPY test /test
WORKDIR /test
RUN mvn verify

# Build the final image
FROM jetty:9.4.26-jre8
COPY --from=0 /test/target/test-1.0-SNAPSHOT.war /var/lib/jetty/webapps/test.war

HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost:8080/test/helloworld
EXPOSE 8080
