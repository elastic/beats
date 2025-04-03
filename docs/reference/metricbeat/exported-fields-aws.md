---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-aws.html
---

# AWS fields [exported-fields-aws]

`aws` module collects AWS monitoring metrics from AWS Cloudwatch.


## aws [_aws]

**`aws.tags.*`**
:   Tag key value pairs from aws resources.

type: object


**`aws.s3.bucket.name`**
:   Name of a S3 bucket.

type: keyword


**`aws.dimensions.*`**
:   Metric dimensions.

type: object


**`aws.*.metrics.*.*`**
:   Metrics that returned from Cloudwatch API query.

type: object


**`aws.linked_account.id`**
:   ID used to identify linked account.

type: keyword


**`aws.linked_account.name`**
:   Name or alias used to identify linked account.

type: keyword



## awshealth [_awshealth]

AWS Health metrics

**`aws.awshealth.affected_entities_others`**
:   The number of affected resources related to the event whose status cannot be verified.

type: float


**`aws.awshealth.affected_entities_pending`**
:   The number of affected resources that may require action.

type: float


**`aws.awshealth.affected_entities_resolved`**
:   The number of affected resources that do not require any action.

type: float


**`aws.awshealth.end_time`**
:   The date and time when the event ended. Some events may not have an end date.

type: date


**`aws.awshealth.event_arn`**
:   The unique identifier for the event. The event ARN has the format arn:aws:health:event-region::event/SERVICE/EVENT_TYPE_CODE/EVENT_TYPE_PLUS_ID.

type: keyword


**`aws.awshealth.event_scope_code`**
:   This parameter specifies whether the Health event is a public Amazon Web Service event or an account-specific event. Allowed values are PUBLIC, ACCOUNT_SPECIFIC, or NONE.

type: keyword


**`aws.awshealth.event_type_category`**
:   The event type category code. Possible values are issue, accountNotification, or scheduledChange.

type: keyword


**`aws.awshealth.event_type_code`**
:   The unique identifier for the event type. The format is AWS_SERVICE_DESCRIPTION.

type: keyword


**`aws.awshealth.last_updated_time`**
:   The most recent date and time when the event was updated.

type: date


**`aws.awshealth.region`**
:   The Amazon Web Services Region name of the event.

type: keyword


**`aws.awshealth.service`**
:   The Amazon Web Service affected by the event. For example, EC2 or RDS.

type: keyword


**`aws.awshealth.start_time`**
:   The date and time when the event began.

type: date


**`aws.awshealth.status_code`**
:   The most recent status of the event. Possible values are open, closed, and upcoming.

type: keyword


**`aws.awshealth.event_description`**
:   The detailed description of the event.

type: text


**`aws.awshealth.affected_entities`**
:   Information about an entity affected by a AWS Health event.

type: array


**`aws.awshealth.affected_entities.aws_account_id`**
:   The Amazon Web Services account number that contains the affected entity.

type: keyword


**`aws.awshealth.affected_entities.entity_url`**
:   The URL of the affected entity.

type: keyword


**`aws.awshealth.affected_entities.entity_value`**
:   The ID of the affected entity.

type: keyword


**`aws.awshealth.affected_entities.last_updated_time`**
:   The most recent time that the entity was updated.

type: date


**`aws.awshealth.affected_entities.status_code`**
:   The most recent status of the event. Possible values are open, closed, and upcoming.

type: keyword


**`aws.awshealth.affected_entities.entity_arn`**
:   The unique identifier for the entity. The entity ARN has the format: arn:aws:health:entity-region:aws-account:entity/entity-id.

type: keyword



## billing [_billing_4]

`billing` contains the estimated charges for your AWS account in Cloudwatch.

**`aws.billing.EstimatedCharges`**
:   Maximum estimated charges for AWS acccount.

type: long


**`aws.billing.Currency`**
:   Estimated charges currency unit.

type: keyword


**`aws.billing.ServiceName`**
:   Service name for the maximum estimated charges.

type: keyword


**`aws.billing.AmortizedCost.amount`**
:   Amortized cost amount

type: double


**`aws.billing.AmortizedCost.unit`**
:   Amortized cost unit

type: keyword


**`aws.billing.BlendedCost.amount`**
:   Blended cost amount

type: double


**`aws.billing.BlendedCost.unit`**
:   Blended cost unit

type: keyword


**`aws.billing.NormalizedUsageAmount.amount`**
:   Normalized usage amount

type: double


**`aws.billing.NormalizedUsageAmount.unit`**
:   Normalized usage amount unit

type: keyword


**`aws.billing.UnblendedCost.amount`**
:   Unblended cost amount

type: double


**`aws.billing.UnblendedCost.unit`**
:   Unblended cost unit

type: keyword


**`aws.billing.UsageQuantity.amount`**
:   Usage quantity amount

type: double


**`aws.billing.UsageQuantity.unit`**
:   Usage quantity unit

type: keyword


**`aws.billing.start_date`**
:   Start date for retrieving AWS costs

type: keyword


**`aws.billing.end_date`**
:   End date for retrieving AWS costs

type: keyword


**`aws.billing.group_definition.key`**
:   The string that represents a key for a specified group

type: keyword


**`aws.billing.group_definition.type`**
:   The string that represents the type of group

type: keyword


**`aws.billing.group_by.*`**
:   Cost explorer group by key values

type: object



## cloudwatch [_cloudwatch_2]

`cloudwatch` contains the metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by different namespaces.

**`aws.cloudwatch.namespace`**
:   The namespace specified when query cloudwatch api.

type: keyword



## dynamodb [_dynamodb_2]

`dynamodb` contains the metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by AWS DynamoDB.

**`aws.dynamodb.metrics.SuccessfulRequestLatency.avg`**
:   The average latency of successful requests to DynamoDB or Amazon DynamoDB Streams during the specified time period.

type: double


**`aws.dynamodb.metrics.SuccessfulRequestLatency.max`**
:   The maximum latency of successful requests to DynamoDB or Amazon DynamoDB Streams during the specified time period.

type: double


**`aws.dynamodb.metrics.OnlineIndexPercentageProgress.avg`**
:   The percentage of completion when a new global secondary index is being added to a table.

type: double


**`aws.dynamodb.metrics.ProvisionedWriteCapacityUnits.avg`**
:   The number of provisioned write capacity units for a table or a global secondary index.

type: double


**`aws.dynamodb.metrics.ProvisionedReadCapacityUnits.avg`**
:   The number of provisioned read capacity units for a table or a global secondary index.

type: double


**`aws.dynamodb.metrics.ConsumedReadCapacityUnits.avg`**
:   The average number of read capacity units consumed over the specified time period, so you can track how much of your provisioned throughput is used.

type: double


**`aws.dynamodb.metrics.ConsumedReadCapacityUnits.sum`**
:   The sum of read capacity units consumed over the specified time period, so you can track how much of your provisioned throughput is used.

type: long


**`aws.dynamodb.metrics.ConsumedWriteCapacityUnits.avg`**
:   The average number of write capacity units consumed over the specified time period, so you can track how much of your provisioned throughput is used.

type: double


**`aws.dynamodb.metrics.ConsumedWriteCapacityUnits.sum`**
:   The sum of write capacity units consumed over the specified time period, so you can track how much of your provisioned throughput is used.

type: long


**`aws.dynamodb.metrics.ReplicationLatency.avg`**
:   The average elapsed time between an updated item appearing in the DynamoDB stream for one replica table, and that item appearing in another replica in the global table.

type: double


**`aws.dynamodb.metrics.ReplicationLatency.max`**
:   The maximum elapsed time between an updated item appearing in the DynamoDB stream for one replica table, and that item appearing in another replica in the global table.

type: double


**`aws.dynamodb.metrics.TransactionConflict.avg`**
:   Average rejected item-level requests due to transactional conflicts between concurrent requests on the same items.

type: double


**`aws.dynamodb.metrics.TransactionConflict.sum`**
:   Total rejected item-level requests due to transactional conflicts between concurrent requests on the same items.

type: long


**`aws.dynamodb.metrics.AccountProvisionedReadCapacityUtilization.avg`**
:   The average percentage of provisioned read capacity units utilized by the account.

type: double


**`aws.dynamodb.metrics.AccountProvisionedWriteCapacityUtilization.avg`**
:   The average percentage of provisioned write capacity units utilized by the account.

type: double


**`aws.dynamodb.metrics.SystemErrors.sum`**
:   The requests to DynamoDB or Amazon DynamoDB Streams that generate an HTTP 500 status code during the specified time period.

type: long


**`aws.dynamodb.metrics.ConditionalCheckFailedRequests.sum`**
:   The number of failed attempts to perform conditional writes.

type: long


**`aws.dynamodb.metrics.PendingReplicationCount.sum`**
:   The number of item updates that are written to one replica table, but that have not yet been written to another replica in the global table.

type: long


**`aws.dynamodb.metrics.ReadThrottleEvents.sum`**
:   Requests to DynamoDB that exceed the provisioned read capacity units for a table or a global secondary index.

type: long


**`aws.dynamodb.metrics.ThrottledRequests.sum`**
:   Requests to DynamoDB that exceed the provisioned throughput limits on a resource (such as a table or an index).

type: long


**`aws.dynamodb.metrics.WriteThrottleEvents.sum`**
:   Requests to DynamoDB that exceed the provisioned write capacity units for a table or a global secondary index.

type: long


**`aws.dynamodb.metrics.AccountMaxReads.max`**
:   The maximum number of read capacity units that can be used by an account. This limit does not apply to on-demand tables or global secondary indexes.

type: long


**`aws.dynamodb.metrics.AccountMaxTableLevelReads.max`**
:   The maximum number of read capacity units that can be used by a table or global secondary index of an account. For on-demand tables this limit caps the maximum read request units a table or a global secondary index can use.

type: long


**`aws.dynamodb.metrics.AccountMaxTableLevelWrites.max`**
:   The maximum number of write capacity units that can be used by a table or global secondary index of an account. For on-demand tables this limit caps the maximum write request units a table or a global secondary index can use.

type: long


**`aws.dynamodb.metrics.AccountMaxWrites.max`**
:   The maximum number of write capacity units that can be used by an account. This limit does not apply to on-demand tables or global secondary indexes.

type: long


**`aws.dynamodb.metrics.MaxProvisionedTableReadCapacityUtilization.max`**
:   The percentage of provisioned read capacity units utilized by the highest provisioned read table or global secondary index of an account.

type: double


**`aws.dynamodb.metrics.MaxProvisionedTableWriteCapacityUtilization.max`**
:   The percentage of provisioned write capacity utilized by the highest provisioned write table or global secondary index of an account.

type: double



## ebs [_ebs_2]

`ebs` contains the metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by AWS EBS.

**`aws.ebs.metrics.VolumeReadBytes.avg`**
:   Average size of each read operation during the period, except on volumes attached to a Nitro-based instance, where the average represents the average over the specified period.

type: double


**`aws.ebs.metrics.VolumeWriteBytes.avg`**
:   Average size of each write operation during the period, except on volumes attached to a Nitro-based instance, where the average represents the average over the specified period.

type: double


**`aws.ebs.metrics.VolumeReadOps.avg`**
:   The total number of read operations in a specified period of time.

type: double


**`aws.ebs.metrics.VolumeWriteOps.avg`**
:   The total number of write operations in a specified period of time.

type: double


**`aws.ebs.metrics.VolumeQueueLength.avg`**
:   The number of read and write operation requests waiting to be completed in a specified period of time.

type: double


**`aws.ebs.metrics.VolumeThroughputPercentage.avg`**
:   The percentage of I/O operations per second (IOPS) delivered of the total IOPS provisioned for an Amazon EBS volume. Used with Provisioned IOPS SSD volumes only.

type: double


**`aws.ebs.metrics.VolumeConsumedReadWriteOps.avg`**
:   The total amount of read and write operations (normalized to 256K capacity units) consumed in a specified period of time. Used with Provisioned IOPS SSD volumes only.

type: double


**`aws.ebs.metrics.BurstBalance.avg`**
:   Used with General Purpose SSD (gp2), Throughput Optimized HDD (st1), and Cold HDD (sc1) volumes only. Provides information about the percentage of I/O credits (for gp2) or throughput credits (for st1 and sc1) remaining in the burst bucket.

type: double


**`aws.ebs.metrics.VolumeTotalReadTime.sum`**
:   The total number of seconds spent by all read operations that completed in a specified period of time.

type: double


**`aws.ebs.metrics.VolumeTotalWriteTime.sum`**
:   The total number of seconds spent by all write operations that completed in a specified period of time.

type: double


**`aws.ebs.metrics.VolumeIdleTime.sum`**
:   The total number of seconds in a specified period of time when no read or write operations were submitted.

type: double



## ec2 [_ec2_2]

`ec2` contains the metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by AWS EC2.

**`aws.ec2.cpu.total.pct`**
:   The percentage of allocated EC2 compute units that are currently in use on the instance.

type: scaled_float


**`aws.ec2.cpu.credit_usage`**
:   The number of CPU credits spent by the instance for CPU utilization.

type: long


**`aws.ec2.cpu.credit_balance`**
:   The number of earned CPU credits that an instance has accrued since it was launched or started.

type: long


**`aws.ec2.cpu.surplus_credit_balance`**
:   The number of surplus credits that have been spent by an unlimited instance when its CPUCreditBalance value is zero.

type: long


**`aws.ec2.cpu.surplus_credits_charged`**
:   The number of spent surplus credits that are not paid down by earned CPU credits, and which thus incur an additional charge.

type: long


**`aws.ec2.network.in.packets`**
:   The total number of packets received on all network interfaces by the instance in collection period.

type: long


**`aws.ec2.network.in.packets_per_sec`**
:   The number of packets per second sent out on all network interfaces by the instance.

type: scaled_float


**`aws.ec2.network.out.packets`**
:   The total number of packets sent out on all network interfaces by the instance in collection period.

type: long


**`aws.ec2.network.out.packets_per_sec`**
:   The number of packets per second sent out on all network interfaces by the instance.

type: scaled_float


**`aws.ec2.network.in.bytes`**
:   The total number of bytes received on all network interfaces by the instance in collection period.

type: long

format: bytes


**`aws.ec2.network.in.bytes_per_sec`**
:   The number of bytes per second received on all network interfaces by the instance.

type: scaled_float


**`aws.ec2.network.out.bytes`**
:   The total number of bytes sent out on all network interfaces by the instance in collection period.

type: long

format: bytes


**`aws.ec2.network.out.bytes_per_sec`**
:   The number of bytes per second sent out on all network interfaces by the instance.

type: scaled_float


**`aws.ec2.diskio.read.bytes`**
:   Total bytes read from all instance store volumes available to the instance in collection period.

type: long

format: bytes


**`aws.ec2.diskio.read.bytes_per_sec`**
:   Bytes read per second from all instance store volumes available to the instance.

type: scaled_float


**`aws.ec2.diskio.write.bytes`**
:   Total bytes written to all instance store volumes available to the instance in collection period.

type: long

format: bytes


**`aws.ec2.diskio.write.bytes_per_sec`**
:   Bytes written per second to all instance store volumes available to the instance.

type: scaled_float


**`aws.ec2.diskio.read.count`**
:   Total completed read operations from all instance store volumes available to the instance in collection period.

type: long


**`aws.ec2.diskio.read.count_per_sec`**
:   Completed read operations per second from all instance store volumes available to the instance in a specified period of time.

type: long


**`aws.ec2.diskio.write.count`**
:   Total completed write operations to all instance store volumes available to the instance in collection period.

type: long


**`aws.ec2.diskio.write.count_per_sec`**
:   Completed write operations per second to all instance store volumes available to the instance in a specified period of time.

type: long


**`aws.ec2.status.check_failed`**
:   Reports whether the instance has passed both the instance status check and the system status check in the last minute.

type: long


**`aws.ec2.status.check_failed_system`**
:   Reports whether the instance has passed the system status check in the last minute.

type: long


**`aws.ec2.status.check_failed_instance`**
:   Reports whether the instance has passed the instance status check in the last minute.

type: long


**`aws.ec2.instance.core.count`**
:   The number of CPU cores for the instance.

type: integer


**`aws.ec2.instance.image.id`**
:   The ID of the image used to launch the instance.

type: keyword


**`aws.ec2.instance.monitoring.state`**
:   Indicates whether detailed monitoring is enabled.

type: keyword


**`aws.ec2.instance.private.dns_name`**
:   The private DNS name of the network interface.

type: keyword


**`aws.ec2.instance.private.ip`**
:   The private IPv4 address associated with the network interface.

type: ip


**`aws.ec2.instance.public.dns_name`**
:   The public DNS name of the instance.

type: keyword


**`aws.ec2.instance.public.ip`**
:   The address of the Elastic IP address (IPv4) bound to the network interface.

type: ip


**`aws.ec2.instance.state.code`**
:   The state of the instance, as a 16-bit unsigned integer.

type: integer


**`aws.ec2.instance.state.name`**
:   The state of the instance (pending | running | shutting-down | terminated | stopping | stopped).

type: keyword


**`aws.ec2.instance.threads_per_core`**
:   The number of threads per CPU core.

type: integer



## elb [_elb_2]

`elb` contains the metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by AWS ELB.

**`aws.elb.metrics.BackendConnectionErrors.sum`**
:   The number of connections that were not successfully established between the load balancer and the registered instances.

type: long


**`aws.elb.metrics.HTTPCode_Backend_2XX.sum`**
:   The number of HTTP 2XX response code generated by registered instances.

type: long


**`aws.elb.metrics.HTTPCode_Backend_3XX.sum`**
:   The number of HTTP 3XX response code generated by registered instances.

type: long


**`aws.elb.metrics.HTTPCode_Backend_4XX.sum`**
:   The number of HTTP 4XX response code generated by registered instances.

type: long


**`aws.elb.metrics.HTTPCode_Backend_5XX.sum`**
:   The number of HTTP 5XX response code generated by registered instances.

type: long


**`aws.elb.metrics.HTTPCode_ELB_4XX.sum`**
:   The number of HTTP 4XX client error codes generated by the load balancer.

type: long


**`aws.elb.metrics.HTTPCode_ELB_5XX.sum`**
:   The number of HTTP 5XX server error codes generated by the load balancer.

type: long


**`aws.elb.metrics.RequestCount.sum`**
:   The number of requests completed or connections made during the specified interval.

type: long


**`aws.elb.metrics.SpilloverCount.sum`**
:   The total number of requests that were rejected because the surge queue is full.

type: long


**`aws.elb.metrics.HealthyHostCount.max`**
:   The number of healthy instances registered with your load balancer.

type: long


**`aws.elb.metrics.SurgeQueueLength.max`**
:   The total number of requests (HTTP listener) or connections (TCP listener) that are pending routing to a healthy instance.

type: long


**`aws.elb.metrics.UnHealthyHostCount.max`**
:   The number of unhealthy instances registered with your load balancer.

type: long


**`aws.elb.metrics.Latency.avg`**
:   The total time elapsed, in seconds, from the time the load balancer sent the request to a registered instance until the instance started to send the response headers.

type: double


**`aws.elb.metrics.EstimatedALBActiveConnectionCount.avg`**
:   The estimated number of concurrent TCP connections active from clients to the load balancer and from the load balancer to targets.

type: double


**`aws.elb.metrics.EstimatedALBConsumedLCUs.avg`**
:   The estimated number of load balancer capacity units (LCU) used by an Application Load Balancer.

type: double


**`aws.elb.metrics.EstimatedALBNewConnectionCount.avg`**
:   The estimated number of new TCP connections established from clients to the load balancer and from the load balancer to targets.

type: double


**`aws.elb.metrics.EstimatedProcessedBytes.avg`**
:   The estimated number of bytes processed by an Application Load Balancer.

type: double



## applicationelb [_applicationelb]

`applicationelb` contains the metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by AWS ApplicationELB.

**`aws.applicationelb.metrics.ActiveConnectionCount.sum`**
:   The total number of concurrent TCP connections active from clients to the load balancer and from the load balancer to targets.

type: long


**`aws.applicationelb.metrics.ClientTLSNegotiationErrorCount.sum`**
:   The number of TLS connections initiated by the client that did not establish a session with the load balancer due to a TLS error.

type: long


**`aws.applicationelb.metrics.HTTP_Fixed_Response_Count.sum`**
:   The number of fixed-response actions that were successful.

type: long


**`aws.applicationelb.metrics.HTTP_Redirect_Count.sum`**
:   The number of redirect actions that were successful.

type: long


**`aws.applicationelb.metrics.HTTP_Redirect_Url_Limit_Exceeded_Count.sum`**
:   The number of redirect actions that couldn’t be completed because the URL in the response location header is larger than 8K.

type: long


**`aws.applicationelb.metrics.HTTPCode_ELB_3XX_Count.sum`**
:   The number of HTTP 3XX redirection codes that originate from the load balancer.

type: long


**`aws.applicationelb.metrics.HTTPCode_ELB_4XX_Count.sum`**
:   The number of HTTP 4XX client error codes that originate from the load balancer.

type: long


**`aws.applicationelb.metrics.HTTPCode_ELB_5XX_Count.sum`**
:   The number of HTTP 5XX server error codes that originate from the load balancer.

type: long


**`aws.applicationelb.metrics.HTTPCode_ELB_500_Count.sum`**
:   The number of HTTP 500 error codes that originate from the load balancer.

type: long


**`aws.applicationelb.metrics.HTTPCode_ELB_502_Count.sum`**
:   The number of HTTP 502 error codes that originate from the load balancer.

type: long


**`aws.applicationelb.metrics.HTTPCode_ELB_503_Count.sum`**
:   The number of HTTP 503 error codes that originate from the load balancer.

type: long


**`aws.applicationelb.metrics.HTTPCode_ELB_504_Count.sum`**
:   The number of HTTP 504 error codes that originate from the load balancer.

type: long


**`aws.applicationelb.metrics.IPv6ProcessedBytes.sum`**
:   The total number of bytes processed by the load balancer over IPv6.

type: long


**`aws.applicationelb.metrics.IPv6RequestCount.sum`**
:   The number of IPv6 requests received by the load balancer.

type: long


**`aws.applicationelb.metrics.NewConnectionCount.sum`**
:   The total number of new TCP connections established from clients to the load balancer and from the load balancer to targets.

type: long


**`aws.applicationelb.metrics.ProcessedBytes.sum`**
:   The total number of bytes processed by the load balancer over IPv4 and IPv6.

type: long


**`aws.applicationelb.metrics.RejectedConnectionCount.sum`**
:   The number of connections that were rejected because the load balancer had reached its maximum number of connections.

type: long


**`aws.applicationelb.metrics.RequestCount.sum`**
:   The number of requests processed over IPv4 and IPv6.

type: long


**`aws.applicationelb.metrics.RuleEvaluations.sum`**
:   The number of rules processed by the load balancer given a request rate averaged over an hour.

type: long


**`aws.applicationelb.metrics.ConsumedLCUs.avg`**
:   The number of load balancer capacity units (LCU) used by your load balancer.

type: double


**`aws.applicationelb.metrics.HealthyHostCount.max`**
:   The number of targets that are considered healthy.

type: long


**`aws.applicationelb.metrics.UnHealthyHostCount.max`**
:   The number of targets that are considered unhealthy.

type: long



## networkelb [_networkelb]

`networkelb` contains the metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by AWS NetworkELB.

**`aws.networkelb.metrics.ActiveFlowCount.avg`**
:   The total number of concurrent flows (or connections) from clients to targets.

type: double


**`aws.networkelb.metrics.ActiveFlowCount_TCP.avg`**
:   The total number of concurrent TCP flows (or connections) from clients to targets.

type: double


**`aws.networkelb.metrics.ActiveFlowCount_TLS.avg`**
:   The total number of concurrent TLS flows (or connections) from clients to targets.

type: double


**`aws.networkelb.metrics.ActiveFlowCount_UDP.avg`**
:   The total number of concurrent UDP flows (or connections) from clients to targets.

type: double


**`aws.networkelb.metrics.ConsumedLCUs.avg`**
:   The number of load balancer capacity units (LCU) used by your load balancer.

type: double


**`aws.networkelb.metrics.ClientTLSNegotiationErrorCount.sum`**
:   The total number of TLS handshakes that failed during negotiation between a client and a TLS listener.

type: long


**`aws.networkelb.metrics.NewFlowCount.sum`**
:   The total number of new flows (or connections) established from clients to targets in the time period.

type: long


**`aws.networkelb.metrics.NewFlowCount_TLS.sum`**
:   The total number of new TLS flows (or connections) established from clients to targets in the time period.

type: long


**`aws.networkelb.metrics.ProcessedBytes.sum`**
:   The total number of bytes processed by the load balancer, including TCP/IP headers.

type: long


**`aws.networkelb.metrics.ProcessedBytes_TLS.sum`**
:   The total number of bytes processed by TLS listeners.

type: long


**`aws.networkelb.metrics.TargetTLSNegotiationErrorCount.sum`**
:   The total number of TLS handshakes that failed during negotiation between a TLS listener and a target.

type: long


**`aws.networkelb.metrics.TCP_Client_Reset_Count.sum`**
:   The total number of reset (RST) packets sent from a client to a target.

type: long


**`aws.networkelb.metrics.TCP_ELB_Reset_Count.sum`**
:   The total number of reset (RST) packets generated by the load balancer.

type: long


**`aws.networkelb.metrics.TCP_Target_Reset_Count.sum`**
:   The total number of reset (RST) packets sent from a target to a client.

type: long


**`aws.networkelb.metrics.HealthyHostCount.max`**
:   The number of targets that are considered healthy.

type: long


**`aws.networkelb.metrics.UnHealthyHostCount.max`**
:   The number of targets that are considered unhealthy.

type: long



## kinesis [_kinesis]

`kinesis` contains the metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by Amazon Kinesis.

**`aws.kinesis.metrics.GetRecords_Bytes.avg`**
:   The average number of bytes retrieved from the Kinesis stream, measured over the specified time period.

type: double


**`aws.kinesis.metrics.GetRecords_IteratorAgeMilliseconds.avg`**
:   The age of the last record in all GetRecords calls made against a Kinesis stream, measured over the specified time period. Age is the difference between the current time and when the last record of the GetRecords call was written to the stream.

type: double


**`aws.kinesis.metrics.GetRecords_Latency.avg`**
:   The time taken per GetRecords operation, measured over the specified time period.

type: double


**`aws.kinesis.metrics.GetRecords_Records.sum`**
:   The number of records retrieved from the shard, measured over the specified time period.

type: long


**`aws.kinesis.metrics.GetRecords_Success.sum`**
:   The number of successful GetRecords operations per stream, measured over the specified time period.

type: long


**`aws.kinesis.metrics.IncomingBytes.avg`**
:   The number of bytes successfully put to the Kinesis stream over the specified time period. This metric includes bytes from PutRecord and PutRecords operations.

type: double


**`aws.kinesis.metrics.IncomingRecords.avg`**
:   The number of records successfully put to the Kinesis stream over the specified time period. This metric includes record counts from PutRecord and PutRecords operations.

type: double


**`aws.kinesis.metrics.PutRecord_Bytes.avg`**
:   The number of bytes put to the Kinesis stream using the PutRecord operation over the specified time period.

type: double


**`aws.kinesis.metrics.PutRecord_Latency.avg`**
:   The time taken per PutRecord operation, measured over the specified time period.

type: double


**`aws.kinesis.metrics.PutRecord_Success.avg`**
:   The percentage of successful writes to a Kinesis stream, measured over the specified time period.

type: double


**`aws.kinesis.metrics.PutRecords_Bytes.avg`**
:   The average number of bytes put to the Kinesis stream using the PutRecords operation over the specified time period.

type: double


**`aws.kinesis.metrics.PutRecords_Latency.avg`**
:   The average time taken per PutRecords operation, measured over the specified time period.

type: double


**`aws.kinesis.metrics.PutRecords_Success.avg`**
:   The total number of PutRecords operations where at least one record succeeded, per Kinesis stream, measured over the specified time period.

type: long


**`aws.kinesis.metrics.PutRecords_TotalRecords.sum`**
:   The total number of records sent in a PutRecords operation per Kinesis data stream, measured over the specified time period.

type: long


**`aws.kinesis.metrics.PutRecords_SuccessfulRecords.sum`**
:   The number of successful records in a PutRecords operation per Kinesis data stream, measured over the specified time period.

type: long


**`aws.kinesis.metrics.PutRecords_FailedRecords.sum`**
:   The number of records rejected due to internal failures in a PutRecords operation per Kinesis data stream, measured over the specified time period.

type: long


**`aws.kinesis.metrics.PutRecords_ThrottledRecords.sum`**
:   The number of records rejected due to throttling in a PutRecords operation per Kinesis data stream, measured over the specified time period.

type: long


**`aws.kinesis.metrics.ReadProvisionedThroughputExceeded.avg`**
:   The number of GetRecords calls throttled for the stream over the specified time period.

type: long


**`aws.kinesis.metrics.SubscribeToShard_RateExceeded.avg`**
:   This metric is emitted when a new subscription attempt fails because there already is an active subscription by the same consumer or if you exceed the number of calls per second allowed for this operation.

type: long


**`aws.kinesis.metrics.SubscribeToShard_Success.avg`**
:   This metric records whether the SubscribeToShard subscription was successfully established.

type: long


**`aws.kinesis.metrics.SubscribeToShardEvent_Bytes.avg`**
:   The number of bytes received from the shard, measured over the specified time period.

type: long


**`aws.kinesis.metrics.SubscribeToShardEvent_MillisBehindLatest.avg`**
:   The difference between the current time and when the last record of the SubscribeToShard event was written to the stream.

type: long


**`aws.kinesis.metrics.SubscribeToShardEvent_Records.sum`**
:   The number of records received from the shard, measured over the specified time period.

type: long


**`aws.kinesis.metrics.SubscribeToShardEvent_Success.avg`**
:   This metric is emitted every time an event is published successfully. It is only emitted when there’s an active subscription.

type: long


**`aws.kinesis.metrics.WriteProvisionedThroughputExceeded.avg`**
:   The number of records rejected due to throttling for the stream over the specified time period. This metric includes throttling from PutRecord and PutRecords operations.

type: long



## lambda [_lambda_2]

`lambda` contains the metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by AWS Lambda.

**`aws.lambda.metrics.Invocations.avg`**
:   The number of times your function code is executed, including successful executions and executions that result in a function error.

type: double


**`aws.lambda.metrics.Errors.avg`**
:   The number of invocations that result in a function error.

type: double


**`aws.lambda.metrics.DeadLetterErrors.avg`**
:   For asynchronous invocation, the number of times Lambda attempts to send an event to a dead-letter queue but fails.

type: double


**`aws.lambda.metrics.DestinationDeliveryFailures.avg`**
:   For asynchronous invocation, the number of times Lambda attempts to send an event to a destination but fails.

type: double


**`aws.lambda.metrics.Duration.avg`**
:   The amount of time that your function code spends processing an event.

type: double


**`aws.lambda.metrics.Throttles.avg`**
:   The number of invocation requests that are throttled.

type: double


**`aws.lambda.metrics.IteratorAge.avg`**
:   For event source mappings that read from streams, the age of the last record in the event.

type: double


**`aws.lambda.metrics.ConcurrentExecutions.avg`**
:   The number of function instances that are processing events.

type: double


**`aws.lambda.metrics.UnreservedConcurrentExecutions.avg`**
:   For an AWS Region, the number of events that are being processed by functions that don’t have reserved concurrency.

type: double


**`aws.lambda.metrics.ProvisionedConcurrentExecutions.max`**
:   The number of function instances that are processing events on provisioned concurrency.

type: long


**`aws.lambda.metrics.ProvisionedConcurrencyUtilization.max`**
:   For a version or alias, the value of ProvisionedConcurrentExecutions divided by the total amount of provisioned concurrency allocated.

type: long


**`aws.lambda.metrics.ProvisionedConcurrencyInvocations.sum`**
:   The number of times your function code is executed on provisioned concurrency.

type: long


**`aws.lambda.metrics.ProvisionedConcurrencySpilloverInvocations.sum`**
:   The number of times your function code is executed on standard concurrency when all provisioned concurrency is in use.

type: long



## natgateway [_natgateway_2]

`natgateway` contains the metrics from Cloudwatch to track usage of NAT gateway related resources.

**`aws.natgateway.metrics.BytesInFromDestination.sum`**
:   The number of bytes received by the NAT gateway from the destination.

type: long


**`aws.natgateway.metrics.BytesInFromSource.sum`**
:   The number of bytes received by the NAT gateway from clients in your VPC.

type: long


**`aws.natgateway.metrics.BytesOutToDestination.sum`**
:   The number of bytes sent out through the NAT gateway to the destination.

type: long


**`aws.natgateway.metrics.BytesOutToSource.sum`**
:   The number of bytes sent through the NAT gateway to the clients in your VPC.

type: long


**`aws.natgateway.metrics.ConnectionAttemptCount.sum`**
:   The number of connection attempts made through the NAT gateway.

type: long


**`aws.natgateway.metrics.ConnectionEstablishedCount.sum`**
:   The number of connections established through the NAT gateway.

type: long


**`aws.natgateway.metrics.ErrorPortAllocation.sum`**
:   The number of times the NAT gateway could not allocate a source port.

type: long


**`aws.natgateway.metrics.IdleTimeoutCount.sum`**
:   The number of connections that transitioned from the active state to the idle state.

type: long


**`aws.natgateway.metrics.PacketsDropCount.sum`**
:   The number of packets dropped by the NAT gateway.

type: long


**`aws.natgateway.metrics.PacketsInFromDestination.sum`**
:   The number of packets received by the NAT gateway from the destination.

type: long


**`aws.natgateway.metrics.PacketsInFromSource.sum`**
:   The number of packets received by the NAT gateway from clients in your VPC.

type: long


**`aws.natgateway.metrics.PacketsOutToDestination.sum`**
:   The number of packets sent out through the NAT gateway to the destination.

type: long


**`aws.natgateway.metrics.PacketsOutToSource.sum`**
:   The number of packets sent through the NAT gateway to the clients in your VPC.

type: long


**`aws.natgateway.metrics.ActiveConnectionCount.max`**
:   The total number of concurrent active TCP connections through the NAT gateway.

type: long



## rds [_rds_2]

`rds` contains the metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by AWS RDS.

**`aws.rds.burst_balance.pct`**
:   The percent of General Purpose SSD (gp2) burst-bucket I/O credits available.

type: scaled_float

format: percent


**`aws.rds.cpu.total.pct`**
:   CPU utilization with value range from 0 to 1.

type: scaled_float

format: percent


**`aws.rds.cpu.credit_usage`**
:   The number of CPU credits spent by the instance for CPU utilization.

type: long


**`aws.rds.cpu.credit_balance`**
:   The number of earned CPU credits that an instance has accrued since it was launched or started.

type: long


**`aws.rds.database_connections`**
:   The number of database connections in use.

type: long


**`aws.rds.db_instance.arn`**
:   Amazon Resource Name(ARN) for each rds.

type: keyword


**`aws.rds.db_instance.class`**
:   Contains the name of the compute and memory capacity class of the DB instance.

type: keyword


**`aws.rds.db_instance.identifier`**
:   Contains a user-supplied database identifier. This identifier is the unique key that identifies a DB instance.

type: keyword


**`aws.rds.db_instance.status`**
:   Specifies the current state of this database.

type: keyword


**`aws.rds.disk_queue_depth`**
:   The number of outstanding IOs (read/write requests) waiting to access the disk.

type: float


**`aws.rds.failed_sql_server_agent_jobs`**
:   The number of failed SQL Server Agent jobs during the last minute.

type: long


**`aws.rds.freeable_memory.bytes`**
:   The amount of available random access memory.

type: long

format: bytes


**`aws.rds.free_storage.bytes`**
:   The amount of available storage space.

type: long

format: bytes


**`aws.rds.maximum_used_transaction_ids`**
:   The maximum transaction ID that has been used. Applies to PostgreSQL.

type: long


**`aws.rds.oldest_replication_slot_lag.mb`**
:   The lagging size of the replica lagging the most in terms of WAL data received. Applies to PostgreSQL.

type: long


**`aws.rds.read.iops`**
:   The average number of disk read I/O operations per second.

type: float


**`aws.rds.replica_lag.sec`**
:   The amount of time a Read Replica DB instance lags behind the source DB instance. Applies to MySQL, MariaDB, and PostgreSQL Read Replicas.

type: long

format: duration


**`aws.rds.swap_usage.bytes`**
:   The amount of swap space used on the DB instance. This metric is not available for SQL Server.

type: long

format: bytes


**`aws.rds.transaction_logs_generation`**
:   The disk space used by transaction logs. Applies to PostgreSQL.

type: long


**`aws.rds.write.iops`**
:   The average number of disk write I/O operations per second.

type: float


**`aws.rds.queries`**
:   The average number of queries executed per second.

type: long


**`aws.rds.deadlocks`**
:   The average number of deadlocks in the database per second.

type: long


**`aws.rds.volume_used.bytes`**
:   The amount of storage used by your Aurora DB instance, in bytes.

type: long

format: bytes


**`aws.rds.volume.read.iops`**
:   The number of billed read I/O operations from a cluster volume, reported at 5-minute intervals.

type: long

format: bytes


**`aws.rds.volume.write.iops`**
:   The number of write disk I/O operations to the cluster volume, reported at 5-minute intervals.

type: long

format: bytes


**`aws.rds.free_local_storage.bytes`**
:   The amount of storage available for temporary tables and logs, in bytes.

type: long

format: bytes


**`aws.rds.login_failures`**
:   The average number of failed login attempts per second.

type: long


**`aws.rds.throughput.commit`**
:   The average number of commit operations per second.

type: float


**`aws.rds.throughput.delete`**
:   The average number of delete queries per second.

type: float


**`aws.rds.throughput.ddl`**
:   The average number of DDL requests per second.

type: float


**`aws.rds.throughput.dml`**
:   The average number of inserts, updates, and deletes per second.

type: float


**`aws.rds.throughput.insert`**
:   The average number of insert queries per second.

type: float


**`aws.rds.throughput.network`**
:   The amount of network throughput both received from and transmitted to clients by each instance in the Aurora MySQL DB cluster, in bytes per second.

type: float


**`aws.rds.throughput.network_receive`**
:   The incoming (Receive) network traffic on the DB instance, including both customer database traffic and Amazon RDS traffic used for monitoring and replication.

type: float


**`aws.rds.throughput.network_transmit`**
:   The outgoing (Transmit) network traffic on the DB instance, including both customer database traffic and Amazon RDS traffic used for monitoring and replication.

type: float


**`aws.rds.throughput.read`**
:   The average amount of time taken per disk I/O operation.

type: float


**`aws.rds.throughput.select`**
:   The average number of select queries per second.

type: float


**`aws.rds.throughput.update`**
:   The average number of update queries per second.

type: float


**`aws.rds.throughput.write`**
:   The average number of bytes written to disk per second.

type: float


**`aws.rds.latency.commit`**
:   The amount of latency for commit operations, in milliseconds.

type: float

format: duration


**`aws.rds.latency.ddl`**
:   The amount of latency for data definition language (DDL) requests, in milliseconds.

type: float

format: duration


**`aws.rds.latency.dml`**
:   The amount of latency for inserts, updates, and deletes, in milliseconds.

type: float

format: duration


**`aws.rds.latency.insert`**
:   The amount of latency for insert queries, in milliseconds.

type: float

format: duration


**`aws.rds.latency.read`**
:   The average amount of time taken per disk I/O operation.

type: float

format: duration


**`aws.rds.latency.select`**
:   The amount of latency for select queries, in milliseconds.

type: float

format: duration


**`aws.rds.latency.update`**
:   The amount of latency for update queries, in milliseconds.

type: float

format: duration


**`aws.rds.latency.write`**
:   The average amount of time taken per disk I/O operation.

type: float

format: duration


**`aws.rds.latency.delete`**
:   The amount of latency for delete queries, in milliseconds.

type: float

format: duration


**`aws.rds.disk_usage.bin_log.bytes`**
:   The amount of disk space occupied by binary logs on the master. Applies to MySQL read replicas.

type: long

format: bytes


**`aws.rds.disk_usage.replication_slot.mb`**
:   The disk space used by replication slot files. Applies to PostgreSQL.

type: long


**`aws.rds.disk_usage.transaction_logs.mb`**
:   The disk space used by transaction logs. Applies to PostgreSQL.

type: long


**`aws.rds.transactions.active`**
:   The average number of current transactions executing on an Aurora database instance per second.

type: long


**`aws.rds.transactions.blocked`**
:   The average number of transactions in the database that are blocked per second.

type: long


**`aws.rds.db_instance.db_cluster_identifier`**
:   This identifier is the unique key that identifies a DB cluster specifically for Amazon Aurora DB cluster.

type: keyword


**`aws.rds.db_instance.role`**
:   DB roles like WRITER or READER, specifically for Amazon Aurora DB cluster.

type: keyword


**`aws.rds.db_instance.engine_name`**
:   Each DB instance runs a DB engine, like MySQL, MariaDB, PostgreSQL and etc.

type: keyword


**`aws.rds.aurora_bin_log_replica_lag`**
:   The amount of time a replica DB cluster running on Aurora with MySQL compatibility lags behind the source DB cluster.

type: long


**`aws.rds.aurora_global_db.replicated_write_io.bytes`**
:   In an Aurora Global Database, the number of write I/O operations replicated from the primary AWS Region to the cluster volume in a secondary AWS Region.

type: long


**`aws.rds.aurora_global_db.data_transfer.bytes`**
:   In an Aurora Global Database, the amount of redo log data transferred from the master AWS Region to a secondary AWS Region.

type: long


**`aws.rds.aurora_global_db.replication_lag.ms`**
:   For an Aurora Global Database, the amount of lag when replicating updates from the primary AWS Region, in milliseconds.

type: long


**`aws.rds.aurora_replica.lag.ms`**
:   For an Aurora Replica, the amount of lag when replicating updates from the primary instance, in milliseconds.

type: long


**`aws.rds.aurora_replica.lag_max.ms`**
:   The maximum amount of lag between the primary instance and each Aurora DB instance in the DB cluster, in milliseconds.

type: long


**`aws.rds.aurora_replica.lag_min.ms`**
:   The minimum amount of lag between the primary instance and each Aurora DB instance in the DB cluster, in milliseconds.

type: long


**`aws.rds.backtrack_change_records.creation_rate`**
:   The number of backtrack change records created over five minutes for your DB cluster.

type: long


**`aws.rds.backtrack_change_records.stored`**
:   The actual number of backtrack change records used by your DB cluster.

type: long


**`aws.rds.backtrack_window.actual`**
:   The difference between the target backtrack window and the actual backtrack window.

type: long


**`aws.rds.backtrack_window.alert`**
:   The number of times that the actual backtrack window is smaller than the target backtrack window for a given period of time.

type: long


**`aws.rds.storage_used.backup_retention_period.bytes`**
:   The total amount of backup storage in bytes used to support the point-in-time restore feature within the Aurora DB cluster’s backup retention window.

type: long


**`aws.rds.storage_used.snapshot.bytes`**
:   The total amount of backup storage in bytes consumed by all Aurora snapshots for an Aurora DB cluster outside its backup retention window.

type: long


**`aws.rds.cache_hit_ratio.buffer`**
:   The percentage of requests that are served by the buffer cache.

type: long


**`aws.rds.cache_hit_ratio.result_set`**
:   The percentage of requests that are served by the Resultset cache.

type: long


**`aws.rds.engine_uptime.sec`**
:   The amount of time that the instance has been running, in seconds.

type: long


**`aws.rds.rds_to_aurora_postgresql_replica_lag.sec`**
:   The amount of lag in seconds when replicating updates from the primary RDS PostgreSQL instance to other nodes in the cluster.

type: long


**`aws.rds.backup_storage_billed_total.bytes`**
:   The total amount of backup storage in bytes for which you are billed for a given Aurora DB cluster.

type: long


**`aws.rds.aurora_volume_left_total.bytes`**
:   The remaining available space for the cluster volume, measured in bytes.

type: long



## s3_daily_storage [_s3_daily_storage_2]

`s3_daily_storage` contains the daily storage metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by AWS S3.

**`aws.s3_daily_storage.bucket.size.bytes`**
:   The amount of data in bytes stored in a bucket.

type: long

format: bytes


**`aws.s3_daily_storage.number_of_objects`**
:   The total number of objects stored in a bucket for all storage classes.

type: long



## s3_request [_s3_request_2]

`s3_request` contains request metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by AWS S3.

**`aws.s3_request.requests.total`**
:   The total number of HTTP requests made to an Amazon S3 bucket, regardless of type.

type: long


**`aws.s3_request.requests.get`**
:   The number of HTTP GET requests made for objects in an Amazon S3 bucket.

type: long


**`aws.s3_request.requests.put`**
:   The number of HTTP PUT requests made for objects in an Amazon S3 bucket.

type: long


**`aws.s3_request.requests.delete`**
:   The number of HTTP DELETE requests made for objects in an Amazon S3 bucket.

type: long


**`aws.s3_request.requests.head`**
:   The number of HTTP HEAD requests made to an Amazon S3 bucket.

type: long


**`aws.s3_request.requests.post`**
:   The number of HTTP POST requests made to an Amazon S3 bucket.

type: long


**`aws.s3_request.requests.select`**
:   The number of Amazon S3 SELECT Object Content requests made for objects in an Amazon S3 bucket.

type: long


**`aws.s3_request.requests.select_scanned.bytes`**
:   The number of bytes of data scanned with Amazon S3 SELECT Object Content requests in an Amazon S3 bucket.

type: long

format: bytes


**`aws.s3_request.requests.select_returned.bytes`**
:   The number of bytes of data returned with Amazon S3 SELECT Object Content requests in an Amazon S3 bucket.

type: long

format: bytes


**`aws.s3_request.requests.list`**
:   The number of HTTP requests that list the contents of a bucket.

type: long


**`aws.s3_request.downloaded.bytes`**
:   The number bytes downloaded for requests made to an Amazon S3 bucket, where the response includes a body.

type: long

format: bytes


**`aws.s3_request.uploaded.bytes`**
:   The number bytes uploaded that contain a request body, made to an Amazon S3 bucket.

type: long

format: bytes


**`aws.s3_request.errors.4xx`**
:   The number of HTTP 4xx client error status code requests made to an Amazon S3 bucket with a value of either 0 or 1.

type: long


**`aws.s3_request.errors.5xx`**
:   The number of HTTP 5xx server error status code requests made to an Amazon S3 bucket with a value of either 0 or 1.

type: long


**`aws.s3_request.latency.first_byte.ms`**
:   The per-request time from the complete request being received by an Amazon S3 bucket to when the response starts to be returned.

type: long

format: duration


**`aws.s3_request.latency.total_request.ms`**
:   The elapsed per-request time from the first byte received to the last byte sent to an Amazon S3 bucket.

type: long

format: duration



## sns [_sns]

`sns` contains the metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by AWS SNS.

**`aws.sns.metrics.PublishSize.avg`**
:   The size of messages published.

type: double


**`aws.sns.metrics.SMSSuccessRate.avg`**
:   The rate of successful SMS message deliveries.

type: double


**`aws.sns.metrics.NumberOfMessagesPublished.sum`**
:   The number of messages published to your Amazon SNS topics.

type: long


**`aws.sns.metrics.NumberOfNotificationsDelivered.sum`**
:   The number of messages successfully delivered from your Amazon SNS topics to subscribing endpoints.

type: long


**`aws.sns.metrics.NumberOfNotificationsFailed.sum`**
:   The number of messages that Amazon SNS failed to deliver.

type: long


**`aws.sns.metrics.NumberOfNotificationsFilteredOut.sum`**
:   The number of messages that were rejected by subscription filter policies.

type: long


**`aws.sns.metrics.NumberOfNotificationsFilteredOut-InvalidAttributes.sum`**
:   The number of messages that were rejected by subscription filter policies because the messages' attributes are invalid - for example, because the attribute JSON is incorrectly formatted.

type: long


**`aws.sns.metrics.NumberOfNotificationsFilteredOut-NoMessageAttributes.sum`**
:   The number of messages that were rejected by subscription filter policies because the messages have no attributes.

type: long


**`aws.sns.metrics.NumberOfNotificationsRedrivenToDlq.sum`**
:   The number of messages that have been moved to a dead-letter queue.

type: long


**`aws.sns.metrics.NumberOfNotificationsFailedToRedriveToDlq.sum`**
:   The number of messages that couldn’t be moved to a dead-letter queue.

type: long


**`aws.sns.metrics.SMSMonthToDateSpentUSD.sum`**
:   The charges you have accrued since the start of the current calendar month for sending SMS messages.

type: long



## sqs [_sqs_2]

`sqs` contains the metrics that were scraped from AWS CloudWatch which contains monitoring metrics sent by AWS SQS.

**`aws.sqs.oldest_message_age.sec`**
:   The maximum approximate age of the oldest non-deleted message in the queue.

type: long

format: duration


**`aws.sqs.messages.delayed`**
:   TThe number of messages in the queue that are delayed and not available for reading immediately.

type: long


**`aws.sqs.messages.not_visible`**
:   The number of messages that are in flight.

type: long


**`aws.sqs.messages.visible`**
:   The number of messages available for retrieval from the queue.

type: long


**`aws.sqs.messages.deleted`**
:   The total number of messages deleted from the queue.

type: long


**`aws.sqs.messages.received`**
:   The total number of messages returned by calls to the ReceiveMessage action.

type: long


**`aws.sqs.messages.sent`**
:   The total number of messages added to a queue.

type: long


**`aws.sqs.empty_receives`**
:   The total number of ReceiveMessage API calls that did not return a message.

type: long


**`aws.sqs.sent_message_size.bytes`**
:   The size of messages added to a queue.

type: long

format: bytes


**`aws.sqs.queue.name`**
:   SQS queue name

type: keyword



## transitgateway [_transitgateway_2]

`transitgateway` contains the metrics from Cloudwatch to track usage of transit gateway related resources.

**`aws.transitgateway.metrics.BytesIn.sum`**
:   The number of bytes received by the transit gateway.

type: long


**`aws.transitgateway.metrics.BytesOut.sum`**
:   The number of bytes sent from the transit gateway.

type: long


**`aws.transitgateway.metrics.PacketsIn.sum`**
:   The number of packets received by the transit gateway.

type: long


**`aws.transitgateway.metrics.PacketsOut.sum`**
:   The number of packets sent by the transit gateway.

type: long


**`aws.transitgateway.metrics.PacketDropCountBlackhole.sum`**
:   The number of packets dropped because they matched a blackhole route.

type: long


**`aws.transitgateway.metrics.PacketDropCountNoRoute.sum`**
:   The number of packets dropped because they did not match a route.

type: long


**`aws.transitgateway.metrics.BytesDropCountNoRoute.sum`**
:   The number of bytes dropped because they did not match a route.

type: long


**`aws.transitgateway.metrics.BytesDropCountBlackhole.sum`**
:   The number of bytes dropped because they matched a blackhole route.

type: long



## usage [_usage_10]

`usage` contains the metrics from Cloudwatch to track usage of some AWS resources.

**`aws.usage.metrics.CallCount.sum`**
:   The number of specified API operations performed in your account.

type: long


**`aws.usage.metrics.ResourceCount.sum`**
:   The number of the specified resources running in your account. The resources are defined by the dimensions associated with the metric.

type: long



## vpn [_vpn_2]

`vpn` contains the metrics from Cloudwatch to track usage of VPN related resources.

**`aws.vpn.metrics.TunnelState.avg`**
:   The state of the tunnel. For static VPNs, 0 indicates DOWN and 1 indicates UP. For BGP VPNs, 1 indicates ESTABLISHED and 0 is used for all other states.

type: double


**`aws.vpn.metrics.TunnelDataIn.sum`**
:   The bytes received through the VPN tunnel.

type: double


**`aws.vpn.metrics.TunnelDataOut.sum`**
:   The bytes sent through the VPN tunnel.

type: double


