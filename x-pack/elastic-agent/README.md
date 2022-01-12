# Elastic Agent developer docs

The source files for the general Elastic Agent documentation are currently stored
in the [observability-docs](https://github.com/elastic/observability-docs) repo. The following docs are only focused on getting developers started building code for Elastic Agent.

## Testing 

Prerequisites:
- installed [mage](https://github.com/magefile/mage)

### Testing docker container

Running Elastic Agent in a docker container is a common use case. To build the Elastic Agent and create a docker image run the following command:

```
DEV=true SNAPSHOT=true PLATFORMS=linux/amd64 TYPES=docker mage package
```

If you are in the 7.13 branch, this will create the `docker.elastic.co/beats/elastic-agent:7.13.0-SNAPSHOT` image in your local environment. Now you can use this to for example test this container with the stack in elastic-package:

```
elastic-package stack up --version=7.13.0-SNAPSHOT -v
```

Please note that the docker container is built in both standard and 'complete' variants.
The 'complete' variant contains extra files, like the chromium browser, that are too large
for the standard variant.

### Testing Elastic Agent on Kubernetes

#### Prerequisites
- create kubernetes cluster using kind, check [here](https://github.com/elastic/beats/blob/master/metricbeat/module/kubernetes/_meta/test/docs/README.md) for details
- deploy kube-state-metrics, check [here](https://github.com/elastic/beats/blob/master/metricbeat/module/kubernetes/_meta/test/docs/README.md) for details
- deploy required infrastructure:
  - for elastic agent in standalone mode: EK stack or use [elastic cloud](https://cloud.elastic.co), check [here](https://github.com/elastic/beats/blob/master/metricbeat/module/kubernetes/_meta/test/docs/README.md) for details
  - for managed mode: use [elastic cloud](https://cloud.elastic.co) or bring up the stack on docker and then connect docker network with kubernetes kind nodes:
  ```
  elastic-package stack up -d -v
  docker network connect elastic-package-stack_default <kind_container_id>
  ```

1. Build elastic-agent:
```bash
cd x-pack/elastic-agent
DEV=true PLATFORMS=linux/amd64 TYPES=docker mage package
```
2. Build docker image:
```bash
cd build/package/elastic-agent/elastic-agent-linux-amd64.docker/docker-build
docker build -t custom-agent-image .
```
3. Load this image in your kind cluster:
```
kind load docker-image custom-agent-image:latest
```
4. Deploy agent with that image:
- download all-in-ome manifest for elastic-agent in standalone or managed mode, change version if needed
```
ELASTIC_AGENT_VERSION="8.0"
ELASTIC_AGENT_MODE="standalone"     # ELASTIC_AGENT_MODE="managed"
curl -L -O https://raw.githubusercontent.com/elastic/beats/${ELASTIC_AGENT_VERSION}/deploy/kubernetes/elastic-agent-${ELASTIC_AGENT_MODE}-kubernetes.yaml
```
- Modify downloaded manifest:
    - change image name to the one, that was created in the previous step and add `imagePullPolicy: Never`:
    ```
    containers:
      - name: elastic-agent
        image: custom-agent-image:latest
        imagePullPolicy: Never
    ``` 
    - set environment variables accordingly to the used setup.

    Elastic-agent in standalone mode: set `ES_USERNAME`, `ES_PASSWORD`,`ES_HOST`.

    Elastic-agent in managed mode: set `FLEET_URL` and `FLEET_ENROLLMENT_TOKEN`.

- create
```
kubectl apply -f elastic-agent-${ELASTIC_AGENT_MODE}-kubernetes.yaml
```
5. Check status of elastic-agent:
```
kubectl -n kube-system get pods -l app=elastic-agent
```
