---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/configuration-autodiscover.html
---

# Autodiscover [configuration-autodiscover]

When you run applications on containers, they become moving targets to the monitoring system. Autodiscover allows you to track them and adapt settings as changes happen. By defining configuration templates, the autodiscover subsystem can monitor services as they start running.

You define autodiscover settings in the  `heartbeat.autodiscover` section of the `heartbeat.yml` config file. To enable autodiscover, you specify a list of providers.


## Providers [_providers]

Autodiscover providers work by watching for events on the system and translating those events into internal autodiscover events with a common format. When you configure the provider, you can optionally use fields from the autodiscover event to set conditions that, when met, launch specific configurations.

On start, Heartbeat will scan existing containers and launch the proper configs for them. Then it will watch for new start/stop events. This ensures you don’t need to worry about state, but only define your desired configs.


#### Docker [_docker_2]

The Docker autodiscover provider watches for Docker containers to start and stop.

It has the following settings:

`host`
:   (Optional) Docker socket (UNIX or TCP socket). It uses `unix:///var/run/docker.sock` by default.

`ssl`
:   (Optional) SSL configuration to use when connecting to the Docker socket.

`cleanup_timeout`
:   (Optional) Specify the time of inactivity before stopping the running configuration for a container, disabled by default.

`labels.dedot`
:   (Optional) Default to be false. If set to true, replace dots in labels with `_`.

These are the fields available within config templating. The `docker.*` fields will be available on each emitted event. event:

* host
* port
* docker.container.id
* docker.container.image
* docker.container.name
* docker.container.labels

For example:

```yaml
{
  "host": "10.4.15.9",
  "port": 6379,
  "docker": {
    "container": {
      "id": "382184ecdb385cfd5d1f1a65f78911054c8511ae009635300ac28b4fc357ce51"
      "name": "redis",
      "image": "redis:3.2.11",
      "labels": {
        "io.kubernetes.pod.namespace": "default"
        ...
      }
    }
  }
}
```

You can define a set of configuration templates to be applied when the condition matches an event. Templates define a condition to match on autodiscover events, together with the list of configurations to launch when this condition happens.

Conditions match events from the provider. Providers use the same format for [Conditions](/reference/heartbeat/defining-processors.md#conditions) that processors use.

Configuration templates can contain variables from the autodiscover event. They can be accessed under the `data` namespace. For example, with the example event, "`${data.port}`" resolves to `6379`.

Heartbeat supports templates for modules:

```yaml
heartbeat.autodiscover:
  providers:
    - type: docker
      templates:
        - condition:
            contains:
              docker.container.image: redis
          config:
            - type: tcp
              hosts: ["${data.host}:${data.port}"]
              schedule: "@every 1s"
              timeout: 1s
```

This configuration launches a `redis` monitor for all containers running an image with `redis` in the name.


#### Kubernetes [_kubernetes]

The Kubernetes autodiscover provider watches for Kubernetes nodes, pods, services to start, update, and stop.

The `kubernetes` autodiscover provider has the following configuration settings:

`node`
:   (Optional) Specify the node to scope heartbeat to in case it cannot be accurately detected, as when running heartbeat in host network mode.

`namespace`
:   (Optional) Select the namespace from which to collect the events from the resources. If it is not set, the provider collects them from all namespaces. It is unset by default. The namespace configuration only applies to kubernetes resources that are namespace scoped and if `unique` field is set to `false`.

`cleanup_timeout`
:   (Optional) Specify the time of inactivity before stopping the running configuration for a container, disabled by default.

`kube_config`
:   (Optional) Use given config file as configuration for Kubernetes client. If kube_config is not set, KUBECONFIG environment variable will be checked and if not present it will fall back to InCluster.

`kube_client_options`
:   (Optional) Additional options can be configured for Kubernetes client. Currently client QPS and burst are supported, if not set Kubernetes client’s [default QPS and burst](https://pkg.go.dev/k8s.io/client-go/rest#pkg-constants) will be used. Example:

```yaml
      kube_client_options:
        qps: 5
        burst: 10
```

`resource`
:   (Optional) Select the resource to do discovery on. Currently supported Kubernetes resources are `pod`, `service` and `node`. If not configured `resource` defaults to `pod`.

`scope`
:   (Optional) Specify at what level autodiscover needs to be done at. `scope` can either take `node` or `cluster` as values. `node` scope allows discovery of resources in the specified node. `cluster` scope allows cluster wide discovery. Only `pod` and `node` resources can be discovered at node scope.

`add_resource_metadata`
:   (Optional) Specify filters and configration for the extra metadata, that will be added to the event. Configuration parameters:

    * `node` or `namespace`: Specify labels and annotations filters for the extra metadata coming from node and namespace. By default all labels are included while annotations are not. To change default behaviour `include_labels`, `exclude_labels` and `include_annotations` can be defined. Those settings are useful when storing labels and annotations that require special handling to avoid overloading the storage output. Note: wildcards are not supported for those settings. The enrichment of `node` or `namespace` metadata can be individually disabled by setting `enabled: false`.
    * `deployment`: If resource is `pod` and it is created from a `deployment`, by default the deployment name isn’t added, this can be enabled by setting `deployment: true`.
    * `cronjob`: If resource is `pod` and it is created from a `cronjob`, by default the cronjob name isn’t added, this can be enabled by setting `cronjob: true`.

        Example:


```yaml
      add_resource_metadata:
        namespace:
          include_labels: ["namespacelabel1"]
        node:
          include_labels: ["nodelabel2"]
          include_annotations: ["nodeannotation1"]
        # deployment: false
        # cronjob: false
```

`unique`
:   (Optional) Defaults to `false`. Marking an autodiscover provider as unique results into making the provider to enable the provided templates only when it will gain the leader lease. This setting can only be combined with `cluster` scope. When `unique` is enabled, `resource` and `add_resource_metadata` settings are not taken into account.

`leader_lease`
:   (Optional) Defaults to `heartbeat-cluster-leader`. This will be name of the lock lease. One can monitor the status of the lease with `kubectl describe lease beats-cluster-leader`. Different Beats that refer to the same leader lease will be competitors in holding the lease and only one will be elected as leader each time.

`leader_leaseduration`
:   (Optional) Duration that non-leader candidates will wait to force acquire the lease leadership. Defaults to `15s`.

`leader_renewdeadline`
:   (Optional) Duration that the leader will retry refreshing its leadership before giving up. Defaults to `10s`.

`leader_retryperiod`
:   (Optional) Duration that the metricbeat instances running to acquire the lease should wait between tries of actions. Defaults to `2s`.

Configuration templates can contain variables from the autodiscover event. These variables can be accessed under the `data` namespace, e.g. to access Pod IP: `${data.kubernetes.pod.ip}`.

These are the fields available within config templating. The `kubernetes.*` fields will be available on each emitted event:


##### Generic fields: [_generic_fields]

* host


##### Pod specific: [_pod_specific]

| Key | Type | Description |
| --- | --- | --- |
| `port` | `string` | Pod port. If pod has multiple ports exposed should be used `ports.<port-name>` instead |
| `kubernetes.namespace` | `string` | Namespace, where the Pod is running |
| `kubernetes.namespace_uuid` | `string` | UUID of the Namespace, where the Pod is running |
| `kubernetes.namespace_annotations.*` | `object` | Annotations of the Namespace, where the Pod is running. Annotations should be used in not dedoted format, e.g. `kubernetes.namespace_annotations.app.kubernetes.io/name` |
| `kubernetes.pod.name` | `string` | Name of the Pod |
| `kubernetes.pod.uid` | `string` | UID of the Pod |
| `kubernetes.pod.ip` | `string` | IP of the Pod |
| `kubernetes.labels.*` | `object` | Object of the Pod labels. Labels should be used in not dedoted format, e.g. `kubernetes.labels.app.kubernetes.io/name` |
| `kubernetes.annotations.*` | `object` | Object of the Pod annotations. Annotations should be used in not dedoted format, e.g. `kubernetes.annotations.test.io/test` |
| `kubernetes.container.name` | `string` | Name of the container |
| `kubernetes.container.runtime` | `string` | Runtime of the container |
| `kubernetes.container.id` | `string` | ID of the container |
| `kubernetes.container.image` | `string` | Image of the container |
| `kubernetes.node.name` | `string` | Name of the Node |
| `kubernetes.node.uid` | `string` | UID of the Node |
| `kubernetes.node.hostname` | `string` | Hostname of the Node |


##### Node specific: [_node_specific]

| Key | Type | Description |
| --- | --- | --- |
| `kubernetes.labels.*` | `object` | Object of labels of the Node |
| `kubernetes.annotations.*` | `object` | Object of annotations of the Node |
| `kubernetes.node.name` | `string` | Name of the Node |
| `kubernetes.node.uid` | `string` | UID of the Node |
| `kubernetes.node.hostname` | `string` | Hostname of the Node |


##### Service specific: [_service_specific]

| Key | Type | Description |
| --- | --- | --- |
| `port` | `string` | Service port |
| `kubernetes.namespace` | `string` | Namespace of the Service |
| `kubernetes.namespace_uuid` | `string` | UUID of the Namespace of the Service |
| `kubernetes.namespace_annotations.*` | `object` | Annotations of the Namespace of the Service. Annotations should be used in not dedoted format, e.g. `kubernetes.namespace_annotations.app.kubernetes.io/name` |
| `kubernetes.labels.*` | `object` | Object of the Service labels |
| `kubernetes.annotations.*` | `object` | Object of the Service annotations |
| `kubernetes.service.name` | `string` | Name of the Service |
| `kubernetes.service.uid` | `string` | UID of the Service |

If the `include_annotations` config is added to the provider config, then the list of annotations present in the config are added to the event.

If the `include_labels` config is added to the provider config, then the list of labels present in the config will be added to the event.

If the `exclude_labels` config is added to the provider config, then the list of labels present in the config will be excluded from the event.

if the `labels.dedot` config is set to be `true` in the provider config, then `.` in labels will be replaced with `_`. By default it is `true`.

if the `annotations.dedot` config is set to be `true` in the provider config, then `.` in annotations will be replaced with `_`. By default it is `true`.

::::{note}
Starting from 8.6 release `kubernetes.labels.*` used in config templating are not dedoted regardless of `labels.dedot` value. This config parameter only affects the fields added in the final Elasticsearch document. For example, for a pod with label `app.kubernetes.io/name=ingress-nginx` the matching condition should be `condition.equals: kubernetes.labels.app.kubernetes.io/name: "ingress-nginx"`. If `labels.dedot` is set to `true`(default value) the label will be stored in Elasticsearch as `kubernetes.labels.app_kubernetes_io/name`. The same applies for kubernetes annotations.
::::


For example:

```yaml
{
  "host": "172.17.0.21",
  "port": 9090,
  "kubernetes": {
    "container": {
      "id": "bb3a50625c01b16a88aa224779c39262a9ad14264c3034669a50cd9a90af1527",
      "image": "prom/prometheus",
      "name": "prometheus"
    },
    "labels": {
      "project": "prometheus",
      ...
    },
    "namespace": "default",
    "node": {
      "name": "minikube"
    },
    "pod": {
      "name": "prometheus-2657348378-k1pnh"
    }
  },
}
```

Heartbeat supports templates for modules:

```yaml
heartbeat.autodiscover:
  providers:
    - type: kubernetes
      include_annotations: ["prometheus.io.scrape"]
      templates:
        - condition:
            contains:
              kubernetes.annotations.prometheus.io/scrape: "true"
          config:
            - type: http
              hosts: ["${data.host}:${data.port}"]
              schedule: "@every 1s"
              timeout: 1s
```

This configuration launches an `http` module for all containers of pods annotated with `prometheus.io/scrape=true`.


#### Amazon ELBs (Deprecated) [_amazon_elbs_deprecated]

**Note: This provider is now deprecated and will be removed in a future release.**

The Amazon ELB autodiscover provider discovers [ELBs](https://aws.amazon.com/elasticloadbalancing/) and their listeners. This is useful when you don’t want to connect directly to a service, but rather to the ELB fronting a pool of services.

This provider will yield one config block per ELB Listener. So, if you have one ELB exposing both ports 80 and 443, it will generate two configs, one for each port. Keep in mind that the beat will de-duplicate configs. So, if the generated configs are the same only one will actually run.

This provider will load AWS credentials using the standard AWS environment variables and shared credentials files see [Best Practices for Managing AWS Access Keys](https://docs.aws.amazon.com/general/latest/gr/aws-access-keys-best-practices.html) for more information. If you do not wish to use these, you may explicitly set the `access_key_id` and `secret_access_key` variables.

These are the available fields during within config templating. The `aws.elb.*` fields will be available on each emitted event.

* host
* port
* cloud.availability_zone
* cloud.provider
* cloud.region
* aws.elb.listener_arn
* aws.elb.load_balancer_arn
* aws.elb.protocol
* aws.elb.type
* aws.elb.scheme
* aws.elb.availability_zones
* aws.elb.created
* aws.elb.state.code
* aws.elb.state.reason
* aws.elb.ip_address_type
* aws.elb.security_groups
* aws.elb.vpc_id
* aws.elb.ssl_policy

Heartbeat supports templates for modules:

```yaml
heartbeat.autodiscover:
  providers:
  - type: aws_elb
    period: 1m
    regions: ["us-east-1", "us-east-2"]
    access_key_id: my-access-key
    secret_access_key: my-secret-access-key
    templates:
    - condition:
        equals.port: 8080
      config:
      - type: tcp
        hosts: ["${data.host}:${data.port}"]
        schedule: "@every 5s"
        timeout: 1s
```

This configuration launches a `tcp` monitor for all ELBs that have a declared port.

This autodiscover provider takes our standard [AWS credentials options](#aws-credentials-config).


## AWS Credentials Configuration [aws-credentials-config]

To configure AWS credentials, either put the credentials into the Heartbeat configuration, or use a shared credentials file, as shown in the following examples.


### Configuration parameters [_configuration_parameters]

* **access_key_id**: first part of access key.
* **secret_access_key**: second part of access key.
* **session_token**: required when using temporary security credentials.
* **credential_profile_name**: profile name in shared credentials file.
* **shared_credential_file**: directory of the shared credentials file.
* **role_arn**: AWS IAM Role to assume.
* **external_id**: external ID to use when assuming a role in another account, see [the AWS documentation for use of external IDs](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-user_externalid.html).
* **proxy_url**: URL of the proxy to use to connect to AWS web services. The syntax is `http(s)://<IP/Hostname>:<port>`
* **fips_enabled**: Enabling this option instructs Heartbeat to use the FIPS endpoint of a service. All services used by Heartbeat are FIPS compatible except for `tagging` but only certain regions are FIPS compatible. See [https://aws.amazon.com/compliance/fips/](https://aws.amazon.com/compliance/fips/) or the appropriate service page, [https://docs.aws.amazon.com/general/latest/gr/aws-service-information.html](https://docs.aws.amazon.com/general/latest/gr/aws-service-information.html), for a full list of FIPS endpoints and regions.
* **ssl**: This specifies SSL/TLS configuration. If the ssl section is missing, the host’s CAs are used for HTTPS connections. See [SSL](/reference/heartbeat/configuration-ssl.md) for more information.
* **default_region**: Default region to query if no other region is set. Most AWS services offer a regional endpoint that can be used to make requests. Some services, such as IAM, do not support regions. If a region is not provided by any other way (environment variable, credential or instance profile), the value set here will be used.
* **assume_role.duration**: The duration of the requested assume role session. Defaults to 15m when not set. AWS allows a maximum session duration between 1h and 12h depending on your maximum session duration policies.
* **assume_role.expiry_window**: The expiry_window will allow refreshing the session prior to its expiration. This is beneficial to prevent expiring tokens from causing requests to fail with an ExpiredTokenException.


### Supported Formats [_supported_formats]

::::{note}
The examples in this section refer to Metricbeat, but the credential options for authentication with AWS are the same no matter which Beat is being used.
::::


* Use `access_key_id`, `secret_access_key`, and/or `session_token`

Users can either put the credentials into the Metricbeat module configuration or use environment variable `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and/or `AWS_SESSION_TOKEN` instead.

If running on Docker, these environment variables should be added as a part of the docker command. For example, with Metricbeat:

```bash
$ docker run -e AWS_ACCESS_KEY_ID=abcd -e AWS_SECRET_ACCESS_KEY=abcd -d --name=metricbeat --user=root --volume="$(pwd)/metricbeat.aws.yml:/usr/share/metricbeat/metricbeat.yml:ro" docker.elastic.co/beats/metricbeat:7.11.1 metricbeat -e -E cloud.auth=elastic:1234 -E cloud.id=test-aws:1234
```

Sample `metricbeat.aws.yml` looks like:

```yaml
metricbeat.modules:
- module: aws
  period: 5m
  access_key_id: ${AWS_ACCESS_KEY_ID}
  secret_access_key: ${AWS_SECRET_ACCESS_KEY}
  session_token: ${AWS_SESSION_TOKEN}
  metricsets:
    - ec2
```

Environment variables can also be added through a file. For example:

```bash
$ cat env.list
AWS_ACCESS_KEY_ID=abcd
AWS_SECRET_ACCESS_KEY=abcd

$ docker run --env-file env.list -d --name=metricbeat --user=root --volume="$(pwd)/metricbeat.aws.yml:/usr/share/metricbeat/metricbeat.yml:ro" docker.elastic.co/beats/metricbeat:7.11.1 metricbeat -e -E cloud.auth=elastic:1234 -E cloud.id=test-aws:1234
```

* Use `credential_profile_name` and/or `shared_credential_file`

If `access_key_id`, `secret_access_key` and `role_arn` are all not given, then heartbeat will check for `credential_profile_name`. If you use different credentials for different tools or applications, you can use profiles to configure multiple access keys in the same configuration file. If there is no `credential_profile_name` given, the default profile will be used.

`shared_credential_file` is optional to specify the directory of your shared credentials file. If it’s empty, the default directory will be used. In Windows, shared credentials file is at `C:\Users\<yourUserName>\.aws\credentials`. For Linux, macOS or Unix, the file is located at `~/.aws/credentials`. When running as a service, the home path depends on the user that manages the service, so the `shared_credential_file` parameter can be used to avoid ambiguity. Please see [Create Shared Credentials File](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/create-shared-credentials-file.md) for more details.

* Use `role_arn`

`role_arn` is used to specify which AWS IAM role to assume for generating temporary credentials. If `role_arn` is given, heartbeat will check if access keys are given. If not, heartbeat will check for credential profile name. If neither is given, default credential profile will be used. Please make sure credentials are given under either a credential profile or access keys.

If running on Docker, the credential file needs to be provided via a volume mount. For example, with Metricbeat:

```bash
docker run -d --name=metricbeat --user=root --volume="$(pwd)/metricbeat.aws.yml:/usr/share/metricbeat/metricbeat.yml:ro" --volume="/Users/foo/.aws/credentials:/usr/share/metricbeat/credentials:ro" docker.elastic.co/beats/metricbeat:7.11.1 metricbeat -e -E cloud.auth=elastic:1234 -E cloud.id=test-aws:1234
```

Sample `metricbeat.aws.yml` looks like:

```yaml
metricbeat.modules:
- module: aws
  period: 5m
  credential_profile_name: elastic-beats
  shared_credential_file: /usr/share/metricbeat/credentials
  metricsets:
    - ec2
```

```yaml
heartbeat.autodiscover:
  providers:
  - type: aws_elb
    period: 1m
    regions: ["us-east-1", "us-east-2"]
    access_key_id: '<access_key_id>'
    secret_access_key: '<secret_access_key>'
    session_token: '<session_token>'
    templates:
    - type: tcp
      hosts: ["${data.host}:${data.port}"]
      schedule: "@every 5s"
      timeout: 1s
```

or

```yaml
heartbeat.autodiscover:
  providers:
  - type: aws_elb
    period: 1m
    regions: ["us-east-1", "us-east-2"]
    access_key_id: '${AWS_ACCESS_KEY_ID:""}'
    secret_access_key: '${AWS_SECRET_ACCESS_KEY:""}'
    session_token: '${AWS_SESSION_TOKEN:""}'
    templates:
    - type: tcp
      hosts: ["${data.host}:${data.port}"]
      schedule: "@every 5s"
      timeout: 1s
```

* Use shared AWS credentials file

```yaml
heartbeat.autodiscover:
  providers:
  - type: aws_elb
    period: 1m
    regions: ["us-east-1", "us-east-2"]
    credential_profile_name: test-hb
    templates:
    - type: tcp
      hosts: ["${data.host}:${data.port}"]
      schedule: "@every 5s"
      timeout: 1s
```


### AWS Credentials Types [_aws_credentials_types]

There are two different types of AWS credentials can be used: access keys and temporary security credentials.

* Access keys

`AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are the two parts of access keys. They are long-term credentials for an IAM user or the AWS account root user. Please see [AWS Access Keys and Secret Access Keys](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html#access-keys-and-secret-access-keys) for more details.

* IAM role ARN

An IAM role is an IAM identity that you can create in your account that has specific permissions that determine what the identity can and cannot do in AWS. A role does not have standard long-term credentials such as a password or access keys associated with it. Instead, when you assume a role, it provides you with temporary security credentials for your role session. IAM role Amazon Resource Name (ARN) can be used to specify which AWS IAM role to assume to generate temporary credentials. Please see [AssumeRole API documentation](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html) for more details.

Here are the steps to set up IAM role using AWS CLI for Metricbeat. Please replace `123456789012` with your own account ID.

Step 1. Create `example-policy.json` file to include all permissions:

```yaml
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "sqs:ReceiveMessage"
            ],
            "Resource": "*"
        },
        {
            "Sid": "VisualEditor1",
            "Effect": "Allow",
            "Action": "sqs:ChangeMessageVisibility",
            "Resource": "arn:aws:sqs:us-east-1:123456789012:test-fb-ks"
        },
        {
            "Sid": "VisualEditor2",
            "Effect": "Allow",
            "Action": "sqs:DeleteMessage",
            "Resource": "arn:aws:sqs:us-east-1:123456789012:test-fb-ks"
        },
        {
            "Sid": "VisualEditor3",
            "Effect": "Allow",
            "Action": [
                "sts:AssumeRole",
                "sqs:ListQueues",
                "tag:GetResources",
                "ec2:DescribeInstances",
                "cloudwatch:GetMetricData",
                "ec2:DescribeRegions",
                "iam:ListAccountAliases",
                "sts:GetCallerIdentity",
                "cloudwatch:ListMetrics"
            ],
            "Resource": "*"
        }
    ]
}
```

Step 2. Create IAM policy using the `aws iam create-policy` command:

```bash
$ aws iam create-policy --policy-name example-policy --policy-document file://example-policy.json
```

Step 3. Create the JSON file `example-role-trust-policy.json` that defines the trust relationship of the IAM role

```yaml
{
    "Version": "2012-10-17",
    "Statement": {
        "Effect": "Allow",
        "Principal": { "AWS": "arn:aws:iam::123456789012:root" },
        "Action": "sts:AssumeRole"
    }
}
```

Step 4. Create the IAM role and attach the policy:

```bash
$ aws iam create-role --role-name example-role --assume-role-policy-document file://example-role-trust-policy.json
$ aws iam attach-role-policy --role-name example-role --policy-arn "arn:aws:iam::123456789012:policy/example-policy"
```

After these steps are done, IAM role ARN can be used for authentication in Metricbeat `aws` module.

* Temporary security credentials

Temporary security credentials has a limited lifetime and consists of an access key ID, a secret access key, and a security token which typically returned from `GetSessionToken`. MFA-enabled IAM users would need to submit an MFA code while calling `GetSessionToken`. Please see [Temporary Security Credentials](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html) for more details. `sts get-session-token` AWS CLI can be used to generate temporary credentials. For example. with MFA-enabled:

```bash
aws> sts get-session-token --serial-number arn:aws:iam::1234:mfa/your-email@example.com --token-code 456789 --duration-seconds 129600
```

Because temporary security credentials are short term, after they expire, the user needs to generate new ones and modify the aws.yml config file with the new credentials. Unless [live reloading](/reference/metricbeat/_live_reloading.md) feature is enabled for Metricbeat, the user needs to manually restart Metricbeat after updating the config file in order to continue collecting Cloudwatch metrics. This will cause data loss if the config file is not updated with new credentials before the old ones expire. For Metricbeat, we recommend users to use access keys in config file to enable aws module making AWS api calls without have to generate new temporary credentials and update the config frequently.

IAM policy is an entity that defines permissions to an object within your AWS environment. Specific permissions needs to be added into the IAM user’s policy to authorize Metricbeat to collect AWS monitoring metrics. Please see documentation under each metricset for required permissions.



