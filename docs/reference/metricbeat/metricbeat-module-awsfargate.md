---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-awsfargate.html
---

# AWS Fargate module [metricbeat-module-awsfargate]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


Amazon ECS on Fargate provides a method to retrieve various metadata, network metrics, and Docker stats about tasks and containers. This is referred to as the [task metadata endpoint](https://docs.aws.amazon.com/AmazonECS/latest/userguide/task-metadata-endpoint-v4-fargate.md) and this endpoint is available per container.

The environment variable is injected by default into the containers of Amazon ECS tasks on Fargate that use platform version 1.4.0 or later and Amazon ECS tasks on Amazon EC2 that are running at least version 1.39.0 of the Amazon ECS container agent.

The awsfargate module is a Metricbeat module which collects AWS fargate metrics from task metadata endpoint.


## Introduction to AWS ECS and Fargate [_introduction_to_aws_ecs_and_fargate]

Amazon Elastic Container Service (Amazon ECS) is a highly scalable, fast, container management service that makes it easy to run, stop, and manage containers. ECS has two launch types that can define how compute resources will be managed: ECS EC2 and ECS Fargate.

* **ECS EC2**

ECS EC2 launches containers that run on EC2 instances. Users have to manage EC2 instances. Pricing depends on the number of EC2 instances running.

One can monitor these containers by deploying Metricbeat on the corresponding EC2 instances with the Metricbeat Docker module enabled.

In order to achieve this one will need:

1. to ensure access to these EC2 instances using ssh keys coupled with EC2 instances (attach ssh keys on cluster creation using `Key pair` option)
2. to enable shh access for the instances with the proper [inbound rules](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/authorizing-access-to-an-instance.html).

* **ECS Fargate**

ECS Fargate removes the responsibility of provisioning, configuring, and managing the EC2 instances by allowing AWS to manage the EC2 instances. Users only need to specify containers and tasks. Pricing based on the number of tasks.


## Task Metadata Endpoint [_task_metadata_endpoint]

[Task metadata endpoint](https://docs.aws.amazon.com/AmazonECS/latest/userguide/task-metadata-endpoint-v4-fargate.md) returns [Docker stats](https://docs.docker.com/engine/api/v1.30/#operation/ContainerStats) in JSON format for all the containers associated with the task. This endpoint is only available from within the task definition itself, which means Metricbeat needs to be run as a sidecar container within the task definition. Since the metadata endpoint is only accessible from within the Fargate Task, there is no authentication in place.


## Metricsets [_metricsets_8]

Currently, we have `task_stats` metricset in `awsfargate` module.


### `task_stats` [_task_stats]

This metricset collects runtime CPU metrics, disk I/O metrics, memory metrics, network metrics and container metadata from both endpoint `${ECS_CONTAINER_METADATA_URI_V4}/task/stats` and `${ECS_CONTAINER_METADATA_URI_V4}/task`.


### Example configuration [_example_configuration_6]

The AWS Fargate module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: awsfargate
  period: 10s
  metricsets:
    - task_stats
```


### Metricsets [_metricsets_9]

The following metricsets are available:

* [task_stats](/reference/metricbeat/metricbeat-metricset-awsfargate-task_stats.md)


