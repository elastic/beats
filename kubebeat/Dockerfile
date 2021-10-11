FROM debian
COPY ./kubebeat /kubebeat
COPY ./kubebeat.yml /kubebeat.yml
ENTRYPOINT ["/kubebeat"]
CMD ["-e", "-d", "'*'"]
