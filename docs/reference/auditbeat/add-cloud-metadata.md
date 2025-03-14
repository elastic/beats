---
navigation_title: "add_cloud_metadata"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/add-cloud-metadata.html
---

# Add cloud metadata [add-cloud-metadata]


The `add_cloud_metadata` processor enriches each event with instance metadata from the machine’s hosting provider. At startup it will query a list of hosting providers and cache the instance metadata.

The following cloud providers are supported:

* Amazon Web Services (AWS)
* Digital Ocean
* Google Compute Engine (GCE)
* [Tencent Cloud](https://www.qcloud.com/?lang=en) (QCloud)
* Alibaba Cloud (ECS)
* Huawei Cloud (ECS)
* Azure Virtual Machine
* Openstack Nova
* Hetzner Cloud


## Special notes [_special_notes]

`huawei` is an alias for `openstack`. Huawei cloud runs on OpenStack platform, and when viewed from a metadata API standpoint, it is impossible to differentiate it from OpenStack. If you know that your deployments run on Huawei Cloud exclusively, and you wish to have `cloud.provider` value as `huawei`, you can achieve this by overwriting the value using an `add_fields` processor.

The Alibaba Cloud and Tencent cloud providers are disabled by default, because they require to access a remote host. The `providers` setting allows users to select a list of default providers to query.

Cloud providers tend to maintain metadata services compliant with other cloud providers. For example, Openstack supports [EC2 compliant metadat service](https://docs.openstack.org/nova/latest/user/metadata.html#ec2-compatible-metadata). This makes it impossible to differentiate cloud provider (`cloud.provider` property) with auto discovery (when `providers` configuration is omitted). The processor implementation incorporates a priority mechanism where priority is given to some providers over others when there are multiple successful metadata results. Currently, `aws/ec2` and `azure` have priority over any other provider as their metadata retrival rely on SDKs. The expectation here is that SDK methods should fail if run in an environment not configured accordingly (ex:- missing configurations or credentials).


## Configurations [_configurations]

The simple configuration below enables the processor.

```yaml
processors:
  - add_cloud_metadata: ~
```

The `add_cloud_metadata` processor has three optional configuration settings. The first one is `timeout` which specifies the maximum amount of time to wait for a successful response when detecting the hosting provider. The default timeout value is `3s`.

If a timeout occurs then no instance metadata will be added to the events. This makes it possible to enable this processor for all your deployments (in the cloud or on-premise).

The second optional setting is `providers`. The `providers` settings accepts a list of cloud provider names to be used. If `providers` is not configured, then all providers that do not access a remote endpoint are enabled by default. The list of providers may alternatively be configured with the environment variable `BEATS_ADD_CLOUD_METADATA_PROVIDERS`, by setting it to a comma-separated list of provider names.

List of names the `providers` setting supports:

* "alibaba", or "ecs" for the Alibaba Cloud provider (disabled by default).
* "azure" for Azure Virtual Machine (enabled by default). If the virtual machine is part of an AKS managed cluster, the fields `orchestrator.cluster.name` and `orchestrator.cluster.id` can also be retrieved. "TENANT_ID", "CLIENT_ID" and "CLIENT_SECRET" environment variables need to be set for authentication purposes. If not set we fallback to [DefaultAzureCredential](https://learn.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication?tabs=bash#2-authenticate-with-azure) and user can choose different authentication methods (e.g. workload identity).
* "digitalocean" for Digital Ocean (enabled by default).
* "aws", or "ec2" for Amazon Web Services (enabled by default).
* "gcp" for Google Copmute Enging (enabled by default).
* "openstack", "nova", or "huawei" for Openstack Nova (enabled by default).
* "openstack-ssl", or "nova-ssl" for Openstack Nova when SSL metadata APIs are enabled (enabled by default).
* "tencent", or "qcloud" for Tencent Cloud (disabled by default).
* "hetzner" for Hetzner Cloud (enabled by default).

For example, configuration below only utilize `aws` metadata retrival mechanism,

```yaml
processors:
  - add_cloud_metadata:
      providers:
        aws
```

The third optional configuration setting is `overwrite`. When `overwrite` is `true`, `add_cloud_metadata` overwrites existing `cloud.*` fields (`false` by default).

The `add_cloud_metadata` processor supports SSL options to configure the http client used to query cloud metadata. See [SSL](/reference/auditbeat/configuration-ssl.md) for more information.


## Provided metadata [_provided_metadata]

The metadata that is added to events varies by hosting provider. Below are examples for each of the supported providers.

*AWS*

Metadata given below are extracted from [instance identity document](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html),

```json
{
  "cloud": {
    "account.id": "123456789012",
    "availability_zone": "us-east-1c",
    "instance.id": "i-4e123456",
    "machine.type": "t2.medium",
    "image.id": "ami-abcd1234",
    "provider": "aws",
    "region": "us-east-1"
  }
}
```

If the EC2 instance has IMDS enabled and if tags are allowed through IMDS endpoint, the processor will further append tags in metadata. Please refer official documentation on [IMDS endpoint](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html) for further details.

```json
{
  "aws": {
    "tags": {
      "org" : "myOrg",
      "owner": "userID"
    }
  }
}
```

*Digital Ocean*

```json
{
  "cloud": {
    "instance.id": "1234567",
    "provider": "digitalocean",
    "region": "nyc2"
  }
}
```

*GCP*

```json
{
  "cloud": {
    "availability_zone": "us-east1-b",
    "instance.id": "1234556778987654321",
    "machine.type": "f1-micro",
    "project.id": "my-dev",
    "provider": "gcp"
  }
}
```

*Tencent Cloud*

```json
{
  "cloud": {
    "availability_zone": "gz-azone2",
    "instance.id": "ins-qcloudv5",
    "provider": "qcloud",
    "region": "china-south-gz"
  }
}
```

*Alibaba Cloud*

This metadata is only available when VPC is selected as the network type of the ECS instance.

```json
{
  "cloud": {
    "availability_zone": "cn-shenzhen",
    "instance.id": "i-wz9g2hqiikg0aliyun2b",
    "provider": "ecs",
    "region": "cn-shenzhen-a"
  }
}
```

*Azure Virtual Machine*

```json
{
  "cloud": {
    "provider": "azure",
    "instance.id": "04ab04c3-63de-4709-a9f9-9ab8c0411d5e",
    "instance.name": "test-az-vm",
    "machine.type": "Standard_D3_v2",
    "region": "eastus2"
  }
}
```

*Openstack Nova*

```json
{
  "cloud": {
    "instance.name": "test-998d932195.mycloud.tld",
    "instance.id": "i-00011a84",
    "availability_zone": "xxxx-az-c",
    "provider": "openstack",
    "machine.type": "m2.large"
  }
}
```

*Hetzner Cloud*

```json
{
  "cloud": {
    "availability_zone": "hel1-dc2",
    "instance.name": "my-hetzner-instance",
    "instance.id": "111111",
    "provider": "hetzner",
    "region": "eu-central"
  }
}
```

