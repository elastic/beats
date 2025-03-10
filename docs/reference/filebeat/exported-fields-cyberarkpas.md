---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-cyberarkpas.html
---

# CyberArk PAS fields [exported-fields-cyberarkpas]

cyberarkpas fields.


## audit [_audit_2]

Cyberark Privileged Access Security Audit fields.

**`cyberarkpas.audit.action`**
:   A description of the audit record.

type: keyword



## ca_properties [_ca_properties]

Account metadata.

**`cyberarkpas.audit.ca_properties.address`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.cpm_disabled`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.cpm_error_details`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.cpm_status`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.creation_method`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.customer`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.database`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.device_type`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.dual_account_status`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.group_name`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.in_process`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.index`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.last_fail_date`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.last_success_change`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.last_success_reconciliation`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.last_success_verification`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.last_task`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.logon_domain`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.policy_id`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.port`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.privcloud`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.reset_immediately`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.retries_count`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.sequence_id`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.tags`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.user_dn`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.user_name`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.virtual_username`**
:   type: keyword


**`cyberarkpas.audit.ca_properties.other`**
:   type: flattened


**`cyberarkpas.audit.category`**
:   The category name (for category-related operations).

type: keyword


**`cyberarkpas.audit.desc`**
:   A static value that displays a description of the audit codes.

type: keyword



## extra_details [_extra_details]

Specific extra details of the audit records.

**`cyberarkpas.audit.extra_details.ad_process_id`**
:   type: keyword


**`cyberarkpas.audit.extra_details.ad_process_name`**
:   type: keyword


**`cyberarkpas.audit.extra_details.application_type`**
:   type: keyword


**`cyberarkpas.audit.extra_details.command`**
:   type: keyword


**`cyberarkpas.audit.extra_details.connection_component_id`**
:   type: keyword


**`cyberarkpas.audit.extra_details.dst_host`**
:   type: keyword


**`cyberarkpas.audit.extra_details.logon_account`**
:   type: keyword


**`cyberarkpas.audit.extra_details.managed_account`**
:   type: keyword


**`cyberarkpas.audit.extra_details.process_id`**
:   type: keyword


**`cyberarkpas.audit.extra_details.process_name`**
:   type: keyword


**`cyberarkpas.audit.extra_details.protocol`**
:   type: keyword


**`cyberarkpas.audit.extra_details.psmid`**
:   type: keyword


**`cyberarkpas.audit.extra_details.session_duration`**
:   type: keyword


**`cyberarkpas.audit.extra_details.session_id`**
:   type: keyword


**`cyberarkpas.audit.extra_details.src_host`**
:   type: keyword


**`cyberarkpas.audit.extra_details.username`**
:   type: keyword


**`cyberarkpas.audit.extra_details.other`**
:   type: flattened


**`cyberarkpas.audit.file`**
:   The name of the target file.

type: keyword


**`cyberarkpas.audit.gateway_station`**
:   The IP of the web application machine (PVWA).

type: ip


**`cyberarkpas.audit.hostname`**
:   The hostname, in upper case.

type: keyword

example: MY-COMPUTER


**`cyberarkpas.audit.iso_timestamp`**
:   The timestamp, in ISO Timestamp format (RFC 3339).

type: date

example: 2013-06-25 10:47:19+00:00


**`cyberarkpas.audit.issuer`**
:   The Vault user who wrote the audit. This is usually the user who performed the operation.

type: keyword


**`cyberarkpas.audit.location`**
:   The target Location (for Location operations).

type: keyword

Field is not indexed.


**`cyberarkpas.audit.message`**
:   A description of the audit records (same information as in the Desc field).

type: keyword


**`cyberarkpas.audit.message_id`**
:   The code ID of the audit records.

type: keyword


**`cyberarkpas.audit.product`**
:   A static value that represents the product.

type: keyword


**`cyberarkpas.audit.pvwa_details`**
:   Specific details of the PVWA audit records.

type: flattened


**`cyberarkpas.audit.raw`**
:   Raw XML for the original audit record. Only present when XSLT file has debugging enabled.

type: keyword

Field is not indexed.


**`cyberarkpas.audit.reason`**
:   The reason entered by the user.

type: text


**`cyberarkpas.audit.rfc5424`**
:   Whether the syslog format complies with RFC5424.

type: boolean

example: True


**`cyberarkpas.audit.safe`**
:   The name of the target Safe.

type: keyword


**`cyberarkpas.audit.severity`**
:   The severity of the audit records.

type: keyword


**`cyberarkpas.audit.source_user`**
:   The name of the Vault user who performed the operation.

type: keyword


**`cyberarkpas.audit.station`**
:   The IP from where the operation was performed. For PVWA sessions, this will be the real client machine IP.

type: ip


**`cyberarkpas.audit.target_user`**
:   The name of the Vault user on which the operation was performed.

type: keyword


**`cyberarkpas.audit.timestamp`**
:   The timestamp, in MMM DD HH:MM:SS format.

type: keyword

example: Jun 25 10:47:19


**`cyberarkpas.audit.vendor`**
:   A static value that represents the vendor.

type: keyword


**`cyberarkpas.audit.version`**
:   A static value that represents the version of the Vault.

type: keyword


