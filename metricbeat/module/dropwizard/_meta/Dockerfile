ARG DROPWIZARD_MAVEN_VERSION
FROM maven:${DROPWIZARD_MAVEN_VERSION}

# Variables used in pom.xml
ARG DROPWIZARD_VERSION
ENV DROPWIZARD_VERSION=${DROPWIZARD_VERSION}
ARG DROPWIZARD_GROUPID
ENV DROPWIZARD_GROUPID=${DROPWIZARD_GROUPID}

COPY test /test

HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost:8080/test/helloworld
EXPOSE 8080

WORKDIR /test

# Install dependencies when building so they are stored in the image
# Change path of local repository because default one is in a volume
ENV MAVEN_OPTS="-Dmaven.repo.local=/opt/m2-repository"
RUN mvn ${MAVEN_OPTS} verify dependency:go-offline

CMD mvn ${MAVEN_OPTS} --offline jetty:run
