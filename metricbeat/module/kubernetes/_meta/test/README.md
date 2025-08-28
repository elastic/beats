# Kube State Metrics metrics files

Each KSM metrics file used for the tests of each `state_*` metricset can be found inside `KSM` directory. Each `state_*` metricset directory will have corresponding `_meta/test*` subfolder with expected files.

Each file has the name format `ksm.v<version>.plain`. The `<version>` should be compatible with the Kubernetes versions we support. Check the compatibility in the official repository [here](https://github.com/kubernetes/kube-state-metrics#compatibility-matrix).

It's mandatory for the name of these files to end with the suffix `.plain`. The reason for this is that the `config.yml` file inside each `state_*` metricset `testdata` requires the suffix `.plain` for the metrics files:

```yaml
type: http
url: "/metrics"
suffix: plain
path: "../_meta/test/KSM"
```

In order to support a new KSM version, first update the KSM version in the `kubernetes.yml` file from the root of the kubernetes module, then apply the file into an existing cluster:

```bash
kubectl apply -f kubernetes.yml
```

After that, you should have a kube-state-metrics pod running. In order to fetch metrics first use port-forward to expose the KSM api:

```bash
kubectl port-forward svc/kube-state-metrics 8080
```

Then you can fetch the metrics from `localhost:8080/metrics` and then save it to a new `./KSM/ksm.vx.xx.x.plain` file.

To generate and check the expectation files, you can run the following commands:

```bash
cd metricbeat/module/kubernetes
# generate the expected files
go test ./state... --data
# test the expected files
go test ./state...
```

> **_NOTE:_**  The expected files inside the two folders of each `state_*` mericset (`_meta/test` and `_meta/testdata`) are not deleted when running the tests. Remember to delete them if they are from an old version.


Since all metricsets use a `data.json` file for the documentation - this file contains an example of the metrics fields -, it is necessary that one of these files has the name `docs.plain`. This file needs to be inside `kubernetes/_meta/test` directory. This file should have the same content as one of the metrics file of a KSM version we support.
