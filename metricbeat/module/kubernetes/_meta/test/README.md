# Kube State Metrics metrics files

Each KSM metrics files has the **name format `ksm.v<version>.plain`**. **The `<version>` should be compatible with the Kubernetes versions we support**. Check the compatibility in the official repository [here](https://github.com/kubernetes/kube-state-metrics#compatibility-matrix).

**These files are being used in the metricsets that fetch these metrics**: all the `state_*` ones. As of the time of this commit (21.feb.2023), there are two folders that are in use for the `test` file present in each metricset: `test` and `testdata`. Both **these folders require these KSM metrics files** to generate the expected ones. You can check the test file by running `go test -data` to generate the expected files, or simply `go test .` to check against the already present generated files.

> **_NOTE:_**  The expected files inside these two folders are not deleted when running the tests. Remember to delete them if they are from an old version.

**The name of these files needs to end with the suffix `.plain`**. The reason for this is that the `config.yml` file inside each `state_*` metricset `testdata` requires the suffix `.plain` for the metrics files:

```yaml
type: http
url: "/metrics"
suffix: plain
path: "../_meta/test"
```

Since all metricsets use a `data.json` file for the documentation - this file contains an example of the metrics fields -, **it is necessary that one of these files has the name `docs.plain`**. This file should have the **same content as one of the metrics file of a KSM version** we support. This means that one of the files is duplicated, for example: `ksm.v2.7.0.plain` has the same content as `docs.plain`. This is not a mistake, as having the `ksm.v2.7.0.plain` tells us the metrics of that specific version.
