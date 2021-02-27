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

## Setup environment for manual tests

Probably the easiest way of testing apiserver is creating a cluster (kind, minikube?), configuring kubeconfig and then

```bash
kubectl proxy --port 8000
```

Metrics for apiserver will be available at `http://localhost:8000/metrics`
