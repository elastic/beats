Filebeat module for AWS ELB
===

Module for the AWS load balancers, it supports the following flavours:

* ELB (Classic Load Balancer)
* Application Load Balancer (V2 Load Balancer for HTTP)
* Network Load Balancer (V2 Load Balancer for TCP and UDP - UDP not tested)

Implementation based on the description of the access logs from the
documentation that can be found in:

* ELB: https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/access-log-collection.html
* Application LB: https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-access-logs.html
* Network LB: https://docs.aws.amazon.com/elasticloadbalancing/latest/network/load-balancer-access-logs.html

Test files starting with `example` are copied or based on examples of this
documentation.


How to manual test this module
===

* Create an ELB and enable access logs for it. A terraform scenario is included
  as example, read the section below about this if you want to use it.
* Make some requests to the load balancer.
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


Using terraform to deploy a testing scenario
====

Terraform configuration is included in the metricset to deploy an scenario that deploys
some instances with running services and a set of load balancers for these
services.

Configuration files can be found in `_meta/terraform`, and deployed with
`terraform apply`. It will get credentials from your configuration, some
settings can be overriden using Terraform variables (see `vars.tf` file).

Once deployed, information about the resources can be queried with `terraform
output`, for example to query the different load balancers: 
  * ELB (classic) load balancer, HTTP listener: `curl $(terraform output elb_http_address)/`
  * ELB (classic) load balancer, TCP listener: `curl $(terraform output elb_tcp_address)/`
  * Application Load Balancer (HTTP): `curl $(terraform output lb_http_address)/`
  * Application Load Balancer (TCP): `curl $(terraform output lb_tcp_address)/`

SQS queue URL needed for configuration of filebeat can be obtained with
`terraform output sqs_queue_url`.

Remember to remove the scenario when not needed with `terraform destroy`.
