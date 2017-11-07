FROM maven:3.3-jdk-8
COPY test /test

HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost:8080/test/helloworld
EXPOSE 8080

WORKDIR /test
CMD mvn jetty:run
