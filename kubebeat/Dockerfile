# Todo delete before merge to elastic/beats
FROM debian
RUN apt-get update
RUN apt-get install -y ca-certificates
COPY ./kubebeat /kubebeat
ENTRYPOINT ["/kubebeat"]
CMD ["-e", "-d", "'*'"]
