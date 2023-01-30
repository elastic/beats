## Testing Metricbeat

### Prerequisites

Make sure that the fields are update. If you don't know how, go to metricbeat directory and run:
```bash
make update
 ```

Check that the modules pass the tests. Go to the directory of each of them and run:
```bash
go test -data
 ```


### Deploy metricbeat
1. Spin up the elastic stack:
   ```bash
   elastic-package stack up -v -d
    ```
   > Don't have elastic package installed? Find how [here](https://github.com/elastic/elastic-package/blob/main/README.md).

2. Create cluster:
    ```bash
    kind create cluster --config=config.yaml
    ```
   > Don't have kind installed? Find how [here](https://kind.sigs.k8s.io/docs/user/quick-start/#installation).

3. Set context:
   ```bash
   kubectl cluster-info --context kind-kind
    ```

4. Connect networks:
      ```bash
    docker network connect elastic-package-stack_default kind-control-plane
    ```

5. Deploy manifests:
   - Testing `state_*` metricsets? Run:
   ```bash
   kubectl apply -k with-ksm
    ```
   - **Not** testing `state_*` metricsets? Run:
   ```bash
   kubectl apply -k without-ksm
    ```
   > **Note**: Adjust hosts for elasticsearch and kibana if they are not correct.

6. Go to metricbeat directory and build the metricbeat binary:
    ```bash
    export CGO_ENABLED=0 && GOOS=linux GOARCH=amd64 go build
    ```

7. Copy the metricbeat binary:
    ```bash
   kubectl cp ./metricbeat `kubectl get pod -n kube-system -l k8s-app=metricbeat -o jsonpath='{.items[].metadata.name}'`:/usr/share/metricbeat/ -n kube-system
    ```

8. Get inside the pod:
   ```bash
   kubectl exec `kubectl get pod -n kube-system -l k8s-app=metricbeat -o jsonpath='{.items[].metadata.name}'` -n kube-system -it -- bash
   ```

9. Once inside the pod, run:
   ```bash
   metricbeat -e -c /etc/metricbeat.yml
    ```

> **Note**: In case you want to test an updated metricbeat binary, you should delete the manifests
> and go back to step 5.


# Running integration tests.

Running the integration tests for the kubernetes module has the requirement of:

* docker
* kind
* kubectl

Once those tools are installed it is as simple as:

```
MODULE="kubernetes" mage goIntegTest
```

The integration tester will use the default context from the kubectl configuration defined
in the `KUBECONFIG` environment variable. There is no requirement that the kubernetes even
be local to your development machine, it just needs to be accessible.

If no `KUBECONFIG` is set and `kind` is installed then the runner will use `kind` to create
a local cluster inside your local docker to perform the integration tests inside. The
`kind` cluster will be created and destroy before and after the test. If you would like to
keep the `kind` cluster running after the test has finished you can set `KIND_SKIP_DELETE=1`
inside of your environment.


## Starting Kubernetes clusters in Cloud providers

The `terraform` directory contains terraform configurations to start Kubernetes
clusters in cloud providers.

