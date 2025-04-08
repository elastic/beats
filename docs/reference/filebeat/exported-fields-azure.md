---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-azure.html
---

# Azure fields [exported-fields-azure]

Azure Module


## azure [_azure]

**`azure.subscription_id`**
:   Azure subscription ID

type: keyword


**`azure.correlation_id`**
:   Correlation ID

type: keyword


**`azure.tenant_id`**
:   tenant ID

type: keyword



## resource [_resource]

Resource

**`azure.resource.id`**
:   Resource ID

type: keyword


**`azure.resource.group`**
:   Resource group

type: keyword


**`azure.resource.provider`**
:   Resource type/namespace

type: keyword


**`azure.resource.namespace`**
:   Resource type/namespace

type: keyword


**`azure.resource.name`**
:   Name

type: keyword


**`azure.resource.authorization_rule`**
:   Authorization rule

type: keyword



## activitylogs [_activitylogs]

Fields for Azure activity logs.

**`azure.activitylogs.identity_name`**
:   identity name

type: keyword



## identity [_identity]

Identity


## claims_initiated_by_user [_claims_initiated_by_user]

Claims initiated by user

**`azure.activitylogs.identity.claims_initiated_by_user.name`**
:   Name

type: keyword


**`azure.activitylogs.identity.claims_initiated_by_user.givenname`**
:   Givenname

type: keyword


**`azure.activitylogs.identity.claims_initiated_by_user.surname`**
:   Surname

type: keyword


**`azure.activitylogs.identity.claims_initiated_by_user.fullname`**
:   Fullname

type: keyword


**`azure.activitylogs.identity.claims_initiated_by_user.schema`**
:   Schema

type: keyword


**`azure.activitylogs.identity.claims.*`**
:   Claims

type: object



## authorization [_authorization]

Authorization

**`azure.activitylogs.identity.authorization.scope`**
:   Scope

type: keyword


**`azure.activitylogs.identity.authorization.action`**
:   Action

type: keyword



## evidence [_evidence]

Evidence

**`azure.activitylogs.identity.authorization.evidence.role_assignment_scope`**
:   Role assignment scope

type: keyword


**`azure.activitylogs.identity.authorization.evidence.role_definition_id`**
:   Role definition ID

type: keyword


**`azure.activitylogs.identity.authorization.evidence.role`**
:   Role

type: keyword


**`azure.activitylogs.identity.authorization.evidence.role_assignment_id`**
:   Role assignment ID

type: keyword


**`azure.activitylogs.identity.authorization.evidence.principal_id`**
:   Principal ID

type: keyword


**`azure.activitylogs.identity.authorization.evidence.principal_type`**
:   Principal type

type: keyword


**`azure.activitylogs.tenant_id`**
:   Tenant ID

type: keyword


**`azure.activitylogs.level`**
:   Level

type: long


**`azure.activitylogs.operation_version`**
:   Operation version

type: keyword


**`azure.activitylogs.operation_name`**
:   Operation name

type: keyword


**`azure.activitylogs.result_type`**
:   Result type

type: keyword


**`azure.activitylogs.result_signature`**
:   Result signature

type: keyword


**`azure.activitylogs.category`**
:   Category

type: keyword


**`azure.activitylogs.event_category`**
:   Event Category

type: keyword


**`azure.activitylogs.properties`**
:   Properties

type: flattened



## auditlogs [_auditlogs]

Fields for Azure audit logs.

**`azure.auditlogs.category`**
:   The category of the operation.  Currently, Audit is the only supported value.

type: keyword


**`azure.auditlogs.operation_name`**
:   The operation name

type: keyword


**`azure.auditlogs.operation_version`**
:   The operation version

type: keyword


**`azure.auditlogs.identity`**
:   Identity

type: keyword


**`azure.auditlogs.tenant_id`**
:   Tenant ID

type: keyword


**`azure.auditlogs.result_signature`**
:   Result signature

type: keyword



## properties [_properties]

The audit log properties

**`azure.auditlogs.properties.result`**
:   Log result

type: keyword


**`azure.auditlogs.properties.activity_display_name`**
:   Activity display name

type: keyword


**`azure.auditlogs.properties.result_reason`**
:   Reason for the log result

type: keyword


**`azure.auditlogs.properties.correlation_id`**
:   Correlation ID

type: keyword


**`azure.auditlogs.properties.logged_by_service`**
:   Logged by service

type: keyword


**`azure.auditlogs.properties.operation_type`**
:   Operation type

type: keyword


**`azure.auditlogs.properties.id`**
:   ID

type: keyword


**`azure.auditlogs.properties.activity_datetime`**
:   Activity timestamp

type: date


**`azure.auditlogs.properties.category`**
:   category

type: keyword



## target_resources.* [_target_resources]

Target resources

**`azure.auditlogs.properties.target_resources.*.display_name`**
:   Display name

type: keyword


**`azure.auditlogs.properties.target_resources.*.id`**
:   ID

type: keyword


**`azure.auditlogs.properties.target_resources.*.type`**
:   Type

type: keyword


**`azure.auditlogs.properties.target_resources.*.ip_address`**
:   ip Address

type: keyword


**`azure.auditlogs.properties.target_resources.*.user_principal_name`**
:   User principal name

type: keyword



## modified_properties.* [_modified_properties]

Modified properties

**`azure.auditlogs.properties.target_resources.*.modified_properties.*.new_value`**
:   New value

type: keyword


**`azure.auditlogs.properties.target_resources.*.modified_properties.*.display_name`**
:   Display value

type: keyword


**`azure.auditlogs.properties.target_resources.*.modified_properties.*.old_value`**
:   Old value

type: keyword



## initiated_by [_initiated_by]

Information regarding the initiator


## app [_app]

App

**`azure.auditlogs.properties.initiated_by.app.servicePrincipalName`**
:   Service principal name

type: keyword


**`azure.auditlogs.properties.initiated_by.app.displayName`**
:   Display name

type: keyword


**`azure.auditlogs.properties.initiated_by.app.appId`**
:   App ID

type: keyword


**`azure.auditlogs.properties.initiated_by.app.servicePrincipalId`**
:   Service principal ID

type: keyword



## user [_user]

User

**`azure.auditlogs.properties.initiated_by.user.userPrincipalName`**
:   User principal name

type: keyword


**`azure.auditlogs.properties.initiated_by.user.displayName`**
:   Display name

type: keyword


**`azure.auditlogs.properties.initiated_by.user.id`**
:   ID

type: keyword


**`azure.auditlogs.properties.initiated_by.user.ipAddress`**
:   ip Address

type: keyword



## platformlogs [_platformlogs]

Fields for Azure platform logs.

**`azure.platformlogs.operation_name`**
:   Operation name

type: keyword


**`azure.platformlogs.result_type`**
:   Result type

type: keyword


**`azure.platformlogs.result_signature`**
:   Result signature

type: keyword


**`azure.platformlogs.category`**
:   Category

type: keyword


**`azure.platformlogs.event_category`**
:   Event Category

type: keyword


**`azure.platformlogs.status`**
:   Status

type: keyword


**`azure.platformlogs.ccpNamespace`**
:   ccpNamespace

type: keyword


**`azure.platformlogs.Cloud`**
:   Cloud

type: keyword


**`azure.platformlogs.Environment`**
:   Environment

type: keyword


**`azure.platformlogs.EventTimeString`**
:   EventTimeString

type: keyword


**`azure.platformlogs.Caller`**
:   Caller

type: keyword


**`azure.platformlogs.ScaleUnit`**
:   ScaleUnit

type: keyword


**`azure.platformlogs.ActivityId`**
:   ActivityId

type: keyword


**`azure.platformlogs.identity_name`**
:   Identity name

type: keyword


**`azure.platformlogs.properties`**
:   Event inner properties

type: flattened



## signinlogs [_signinlogs]

Fields for Azure sign-in logs.

**`azure.signinlogs.operation_name`**
:   The operation name

type: keyword


**`azure.signinlogs.operation_version`**
:   The operation version

type: keyword


**`azure.signinlogs.tenant_id`**
:   Tenant ID

type: keyword


**`azure.signinlogs.result_signature`**
:   Result signature

type: keyword


**`azure.signinlogs.result_description`**
:   Result description

type: keyword


**`azure.signinlogs.result_type`**
:   Result type

type: keyword


**`azure.signinlogs.identity`**
:   Identity

type: keyword


**`azure.signinlogs.category`**
:   Category

type: keyword


**`azure.signinlogs.properties.id`**
:   Unique ID representing the sign-in activity.

type: keyword


**`azure.signinlogs.properties.created_at`**
:   Date and time (UTC) the sign-in was initiated.

type: date


**`azure.signinlogs.properties.user_display_name`**
:   User display name

type: keyword


**`azure.signinlogs.properties.correlation_id`**
:   Correlation ID

type: keyword


**`azure.signinlogs.properties.user_principal_name`**
:   User principal name

type: keyword


**`azure.signinlogs.properties.user_id`**
:   User ID

type: keyword


**`azure.signinlogs.properties.app_id`**
:   App ID

type: keyword


**`azure.signinlogs.properties.app_display_name`**
:   App display name

type: keyword


**`azure.signinlogs.properties.autonomous_system_number`**
:   Autonomous system number.

type: long


**`azure.signinlogs.properties.client_app_used`**
:   Client app used

type: keyword


**`azure.signinlogs.properties.conditional_access_status`**
:   Conditional access status

type: keyword


**`azure.signinlogs.properties.original_request_id`**
:   Original request ID

type: keyword


**`azure.signinlogs.properties.is_interactive`**
:   Is interactive

type: boolean


**`azure.signinlogs.properties.token_issuer_name`**
:   Token issuer name

type: keyword


**`azure.signinlogs.properties.token_issuer_type`**
:   Token issuer type

type: keyword


**`azure.signinlogs.properties.processing_time_ms`**
:   Processing time in milliseconds

type: float


**`azure.signinlogs.properties.risk_detail`**
:   Risk detail

type: keyword


**`azure.signinlogs.properties.risk_level_aggregated`**
:   Risk level aggregated

type: keyword


**`azure.signinlogs.properties.risk_level_during_signin`**
:   Risk level during signIn

type: keyword


**`azure.signinlogs.properties.risk_state`**
:   Risk state

type: keyword


**`azure.signinlogs.properties.resource_display_name`**
:   Resource display name

type: keyword


**`azure.signinlogs.properties.status.error_code`**
:   Error code

type: long


**`azure.signinlogs.properties.device_detail.device_id`**
:   Device ID

type: keyword


**`azure.signinlogs.properties.device_detail.operating_system`**
:   Operating system

type: keyword


**`azure.signinlogs.properties.device_detail.browser`**
:   Browser

type: keyword


**`azure.signinlogs.properties.device_detail.display_name`**
:   Display name

type: keyword


**`azure.signinlogs.properties.device_detail.trust_type`**
:   Trust type

type: keyword


**`azure.signinlogs.properties.device_detail.is_compliant`**
:   If the device is compliant

type: boolean


**`azure.signinlogs.properties.device_detail.is_managed`**
:   If the device is managed

type: boolean


**`azure.signinlogs.properties.applied_conditional_access_policies`**
:   A list of conditional access policies that are triggered by the corresponding sign-in activity.

type: array


**`azure.signinlogs.properties.authentication_details`**
:   The result of the authentication attempt and additional details on the authentication method.

type: array


**`azure.signinlogs.properties.authentication_processing_details`**
:   Additional authentication processing details, such as the agent name in case of PTA/PHS or Server/farm name in case of federated authentication.

type: flattened


**`azure.signinlogs.properties.authentication_protocol`**
:   Authentication protocol type.

type: keyword


**`azure.signinlogs.properties.incoming_token_type`**
:   Incoming token type.

type: keyword


**`azure.signinlogs.properties.unique_token_identifier`**
:   Unique token identifier for the request.

type: keyword


**`azure.signinlogs.properties.authentication_requirement`**
:   This holds the highest level of authentication needed through all the sign-in steps, for sign-in to succeed.

type: keyword


**`azure.signinlogs.properties.authentication_requirement_policies`**
:   Set of CA policies that apply to this sign-in, each as CA: policy name, and/or MFA: Per-user

type: flattened


**`azure.signinlogs.properties.flagged_for_review`**
:   type: boolean


**`azure.signinlogs.properties.home_tenant_id`**
:   type: keyword


**`azure.signinlogs.properties.network_location_details`**
:   The network location details including the type of network used and its names.

type: array


**`azure.signinlogs.properties.resource_id`**
:   The identifier of the resource that the user signed in to.

type: keyword


**`azure.signinlogs.properties.resource_tenant_id`**
:   type: keyword


**`azure.signinlogs.properties.risk_event_types`**
:   The list of risk event types associated with the sign-in. Possible values: unlikelyTravel, anonymizedIPAddress, maliciousIPAddress, unfamiliarFeatures, malwareInfectedIPAddress, suspiciousIPAddress, leakedCredentials, investigationsThreatIntelligence, generic, or unknownFutureValue.

type: keyword


**`azure.signinlogs.properties.risk_event_types_v2`**
:   The list of risk event types associated with the sign-in. Possible values: unlikelyTravel, anonymizedIPAddress, maliciousIPAddress, unfamiliarFeatures, malwareInfectedIPAddress, suspiciousIPAddress, leakedCredentials, investigationsThreatIntelligence, generic, or unknownFutureValue.

type: keyword


**`azure.signinlogs.properties.service_principal_name`**
:   The application name used for sign-in. This field is populated when you are signing in using an application.

type: keyword


**`azure.signinlogs.properties.user_type`**
:   type: keyword


**`azure.signinlogs.properties.service_principal_id`**
:   The application identifier used for sign-in. This field is populated when you are signing in using an application.

type: keyword


**`azure.signinlogs.properties.cross_tenant_access_type`**
:   type: keyword


**`azure.signinlogs.properties.is_tenant_restricted`**
:   type: boolean


**`azure.signinlogs.properties.sso_extension_version`**
:   type: keyword


