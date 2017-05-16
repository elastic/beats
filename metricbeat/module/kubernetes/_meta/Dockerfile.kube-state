FROM gcr.io/google_containers/kube-state-metrics:v0.5.0

ADD kubeconfig /

ENTRYPOINT ["/kube-state-metrics"]
CMD ["--port=8080", "--in-cluster=false", "--apiserver=http://172.17.0.1:8080", "--kubeconfig=/kubeconfig"]
