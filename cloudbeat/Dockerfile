# Todo delete before merge to elastic/beats
FROM debian
RUN apt-get update
RUN apt-get install -y ca-certificates
COPY ./cloudbeat /cloudbeat
COPY ./cloudbeat.yml /cloudbeat.yml
ENTRYPOINT ["/cloudbeat"]
CMD ["-e", "-d", "'*'"]
