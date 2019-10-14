Filebeat module for AWS ELB
===

Implementation based on the description of the access logs from the
documentation that can be found in https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/access-log-collection.html

Test files starting with example are copied or based on examples of this
documentation.


How to manual test this module
===

* Create an ELB and enable access logs for it, this can be done using the
  terraform configuration in `_meta`.
* Make some requests to the service, if terraform was used, this can be done
  with:
  * ELB (classic) load balancer: `curl $(terraform output elb_address)/`
  * Application Load Balancer: `curl $(terraform output elb_address)/`
* Configure filebeat, using the queue url from `terraform output sqs_queue_url`.
```
filebeat.modules:
- module: aws
  elb:
   enabled: true
   var.queue_url: <queue url>
   var.credential_profile_name: <profile name>
  s3access:
   enabled: false
```
* Check parsed logs

Please notice that ELB logs can take some minutes before being available in S3.
