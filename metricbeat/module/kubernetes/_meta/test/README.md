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


When you update the KSM directory files, remember to run `go test -data` inside each `state_*` metricset directory to generate the expected files. To check against the expected files already present, you can just run `go test .`.

> **TIP**: To run tests and generate the expected files for all state metricsets you can run `go test ./state... --data`. Navigate to `/elastic/beats/metricbeat/module/kubernetes/` to run this command.

> **_NOTE:_**  The expected files inside the two folders of each `state_*` mericset (`_meta/test` and `_meta/testdata`) are not deleted when running the tests. Remember to delete them if they are from an old version.


Since all metricsets use a `data.json` file for the documentation - this file contains an example of the metrics fields -, it is necessary that one of these files has the name `docs.plain`. This file needs to be inside `kubernetes/_meta/test` directory. This file should have the same content as one of the metrics file of a KSM version we support.
