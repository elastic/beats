Filebeat module for AWS CloudTrail Logs
===

Module for AWS CloudTrail logs which captures information about
actions taken by a user, role or an AWS service.  Events include
actions taken in the AWS Management Console, AWS Command Line
interface and AWS SDKs and APIs. These logs can
help with:

* Governance
* Compliance
* Operational and risk auditing

Implementation based on the description of CloudTrail from the
documentation that can be found in:

* CloudTrail Record Contents: https://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-event-reference-record-contents.html
* CloudTrail Log File Examples: https://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-log-file-examples.html

It should be noted that the `cloudtrail` fileset does not read the
CloudTrail Digest files that are delivered to the S3 bucket when Log
File Integrity is turned on, it only reads the CloudTrail logs.

How to manual test this module
===

* Create a CloudTrail with a S3 bucket as the storage location
* Configure this S3 bucket to send "All object create events" to a SQS queue
* Configure filebeat, using the SQS queue url with s3 notification setup in 
previous step.
```
filebeat.modules:
- module: aws
  cloudtrail:
    enabled: true
    var.queue_url: <queue url>
    var.credential_profile_name: <profile name>
```
* Check parsed logs
