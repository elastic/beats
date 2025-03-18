---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-gcp.html
---

# Google Cloud Platform (GCP) fields [exported-fields-gcp]

Module for handling logs from Google Cloud.


## gcp [_gcp]

Fields from Google Cloud logs.


## destination.instance [_destination_instance]

If the destination of the connection was a VM located on the same VPC, this field is populated with VM instance details. In a Shared VPC configuration, project_id corresponds to the project that owns the instance, usually the service project.

**`gcp.destination.instance.project_id`**
:   ID of the project containing the VM.

type: keyword


**`gcp.destination.instance.region`**
:   Region of the VM.

type: keyword


**`gcp.destination.instance.zone`**
:   Zone of the VM.

type: keyword



## destination.vpc [_destination_vpc]

If the destination of the connection was a VM located on the same VPC, this field is populated with VPC network details. In a Shared VPC configuration, project_id corresponds to that of the host project.

**`gcp.destination.vpc.project_id`**
:   ID of the project containing the VM.

type: keyword


**`gcp.destination.vpc.vpc_name`**
:   VPC on which the VM is operating.

type: keyword


**`gcp.destination.vpc.subnetwork_name`**
:   Subnetwork on which the VM is operating.

type: keyword



## source.instance [_source_instance]

If the source of the connection was a VM located on the same VPC, this field is populated with VM instance details. In a Shared VPC configuration, project_id corresponds to the project that owns the instance, usually the service project.

**`gcp.source.instance.project_id`**
:   ID of the project containing the VM.

type: keyword


**`gcp.source.instance.region`**
:   Region of the VM.

type: keyword


**`gcp.source.instance.zone`**
:   Zone of the VM.

type: keyword



## source.vpc [_source_vpc]

If the source of the connection was a VM located on the same VPC, this field is populated with VPC network details. In a Shared VPC configuration, project_id corresponds to that of the host project.

**`gcp.source.vpc.project_id`**
:   ID of the project containing the VM.

type: keyword


**`gcp.source.vpc.vpc_name`**
:   VPC on which the VM is operating.

type: keyword


**`gcp.source.vpc.subnetwork_name`**
:   Subnetwork on which the VM is operating.

type: keyword



## audit [_audit_3]

Fields for Google Cloud audit logs.

**`gcp.audit.type`**
:   Type property.

type: keyword



## authentication_info [_authentication_info]

Authentication information.

**`gcp.audit.authentication_info.principal_email`**
:   The email address of the authenticated user making the request.

type: keyword


**`gcp.audit.authentication_info.authority_selector`**
:   The authority selector specified by the requestor, if any. It is not guaranteed  that the principal was allowed to use this authority.

type: keyword


**`gcp.audit.authorization_info`**
:   Authorization information for the operation.

type: array


**`gcp.audit.method_name`**
:   The name of the service method or operation. For API calls, this  should be the name of the API method.  For example, *google.datastore.v1.Datastore.RunQuery*.

type: keyword


**`gcp.audit.num_response_items`**
:   The number of items returned from a List or Query API method, if applicable.

type: long



## request [_request]

The operation request.

**`gcp.audit.request.proto_name`**
:   Type property of the request.

type: keyword


**`gcp.audit.request.filter`**
:   Filter of the request.

type: keyword


**`gcp.audit.request.name`**
:   Name of the request.

type: keyword


**`gcp.audit.request.resource_name`**
:   Name of the request resource.

type: keyword



## request_metadata [_request_metadata]

Metadata about the request.

**`gcp.audit.request_metadata.caller_ip`**
:   The IP address of the caller.

type: ip


**`gcp.audit.request_metadata.caller_supplied_user_agent`**
:   The user agent of the caller. This information is not authenticated and  should be treated accordingly.

type: keyword



## response [_response]

The operation response.

**`gcp.audit.response.proto_name`**
:   Type property of the response.

type: keyword



## details [_details]

The details of the response.

**`gcp.audit.response.details.group`**
:   The name of the group.

type: keyword


**`gcp.audit.response.details.kind`**
:   The kind of the response details.

type: keyword


**`gcp.audit.response.details.name`**
:   The name of the response details.

type: keyword


**`gcp.audit.response.details.uid`**
:   The uid of the response details.

type: keyword


**`gcp.audit.response.status`**
:   Status of the response.

type: keyword


**`gcp.audit.resource_name`**
:   The resource or collection that is the target of the operation.  The name is a scheme-less URI, not including the API service name.  For example, *shelves/SHELF_ID/books*.

type: keyword



## resource_location [_resource_location]

The location of the resource.

**`gcp.audit.resource_location.current_locations`**
:   Current locations of the resource.

type: keyword


**`gcp.audit.service_name`**
:   The name of the API service performing the operation.  For example, datastore.googleapis.com.

type: keyword



## status [_status]

The status of the overall operation.

**`gcp.audit.status.code`**
:   The status code, which should be an enum value of google.rpc.Code.

type: integer


**`gcp.audit.status.message`**
:   A developer-facing error message, which should be in English. Any user-facing  error message should be localized and sent in the google.rpc.Status.details  field, or localized by the client.

type: keyword



## firewall [_firewall_2]

Fields for Google Cloud Firewall logs.


## rule_details [_rule_details]

Description of the firewall rule that matched this connection.

**`gcp.firewall.rule_details.priority`**
:   The priority for the firewall rule.

type: long


**`gcp.firewall.rule_details.action`**
:   Action that the rule performs on match.

type: keyword


**`gcp.firewall.rule_details.direction`**
:   Direction of traffic that matches this rule.

type: keyword


**`gcp.firewall.rule_details.reference`**
:   Reference to the firewall rule.

type: keyword


**`gcp.firewall.rule_details.source_range`**
:   List of source ranges that the firewall rule applies to.

type: keyword


**`gcp.firewall.rule_details.destination_range`**
:   List of destination ranges that the firewall applies to.

type: keyword


**`gcp.firewall.rule_details.source_tag`**
:   List of all the source tags that the firewall rule applies to.

type: keyword


**`gcp.firewall.rule_details.target_tag`**
:   List of all the target tags that the firewall rule applies to.

type: keyword


**`gcp.firewall.rule_details.ip_port_info`**
:   List of ip protocols and applicable port ranges for rules.

type: array


**`gcp.firewall.rule_details.source_service_account`**
:   List of all the source service accounts that the firewall rule applies to.

type: keyword


**`gcp.firewall.rule_details.target_service_account`**
:   List of all the target service accounts that the firewall rule applies to.

type: keyword



## vpcflow [_vpcflow_2]

Fields for Google Cloud VPC flow logs.

**`gcp.vpcflow.reporter`**
:   The side which reported the flow. Can be either *SRC* or *DEST*.

type: keyword


**`gcp.vpcflow.rtt.ms`**
:   Latency as measured (for TCP flows only) during the time interval. This is the time elapsed between sending a SEQ and receiving a corresponding ACK and it contains the network RTT as well as the application related delay.

type: long


