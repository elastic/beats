# Kube apiserver Stats

## Version history

- Initial version, `v1.8.8`

    There might be a (non reported yet) issue with this version due to the label `code` missing.
    Beats 7.4 should solve the issue.

- June 2019, `v1.14.3`

    `apiserver_request_total` will be used in spite of `apiserver_request_count`.
    An _ugly trick_ has been put in place that will read both values, using `apiserver_request_total` if exists. The deprecated value is being configured under the bogus name `request.beforev14.count` and renamed to `request.count` if the newer does not exists.

## Resources

`apiserver_request_latencies`
    - component
    - group
    - resource
    - scope
    - subresource
    - verb
    - version

`apiserver_request_duration_seconds_bucket`
    - component
    - dry_run
    - group
    - resource
    - scope
    - subresource
    - verb
    - version

`apiserver_request_total`
    - client
    - code. Note: this one was not being added at previous.
    - component
    - contentType
    - dry_run
    - resource
    - scope
    - subresource
    - verb
    - version

`apiserver_longrunning_gauge`
    - component
    - group
    - resource
    - scope
    - subresource
    - verb
    - version

`etcd_object_counts`
    - resource

`apiserver_current_inflight_requests`
    - requestKind

`apiserver_audit_event_total`

`apiserver_audit_requests_rejected_total`

## Generating expectation files

In order to support a new Kubernetes releases you'll have to generate new expectation files for this module in `_meta/test`. For that, start by deploying a new kubernetes cluster on the required Kubernetes version, for example:

```bash
kind create cluster --image kindest/node:v1.32.0
```

After that, you can apply the [`kubernetes.yml`](https://github.com/elastic/beats/blob/main/metricbeat/module/kubernetes/kubernetes.yml) file from the root of the kubernetes module:

```bash
kubectl apply -f kubernetes.yml
```

This is required since accessing the apiserver metrics requires additional permissions.

Next, expose the apiserver api:

```bash
kubectl port-forward -n kube-system pod/kube-apiserver-kind-control-plane 6443
```

Then you can fetch the metrics from the url provided in the output and save it to a new `_meta/test/metrics.x.xx` file.

```bash
curl -k https://localhost:6443/metrics > _meta/test/metrics.x.xx
```

Run the following commands to generate and test the expected files:

```bash
cd metricbeat/module/kubernetes/apiserver
# generate the expected files
go test ./... --data
# test the expected files
go test ./...
```
