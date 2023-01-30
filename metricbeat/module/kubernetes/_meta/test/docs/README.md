# Testing metricbeat

## Check that all tests succeed

First you need to make sure that the fields are updated. If you don't know how, go to metricbeat directory and run:
```bash
make update
 ```

Check that the modules pass the tests. Go to the directory of each of them and run:
```bash
go test -data
 ```

You can also run the integrations test by applying the following command:
```bash
MODULE="kubernetes" mage goIntegTest
```



## Deploy metricbeat manually
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

3. Set the context:
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

> **Note**: In case you want to test an updated metricbeat binary, you should delete the manifests:
>   `kubectl delete -k with-ksm` or `kubectl delete -k without-ksm`, and go back to step 5.


