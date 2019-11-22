Filebeat module for AWS VPC Logs
===

Module for the AWS virtual private cloud (VPC) logs which captures information
about the IP traffic going to and from network interfaces in VPC. These logs can
help with:

* Diagnosing overly restrictive security group rules
* Monitoring the traffic that is reaching your instance
* Determining the direction of the traffic to and from the network interfaces

Implementation based on the description of the flow logs from the
documentation that can be found in:

* Default Flow Log Format: https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs.html
* Custom Format with Traffic Through a NAT Gateway: https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs-records-examples.html
* Custom Format with Traffic Through a Transit Gateway: https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs-records-examples.html

Test files are copied from examples of these documentation.


How to manual test this module
===

* Create a VPC and enable publishing flow logs to Amazon S3.
* Configure this S3 bucket to publish notifications to a SQS queue in the same 
region when new objects are created.
* Configure filebeat, using the SQS queue url with s3 notification setup in 
previous step.
```
filebeat.modules:
- module: aws
  vpcflow:
    enabled: true
    var.queue_url: <queue url>
    var.credential_profile_name: <profile name>
  s3access:
    enabled: false
  elb:
    enabled: false
```
* Check parsed logs
