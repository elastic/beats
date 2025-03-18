---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-okta.html
---

# Okta fields [exported-fields-okta]

Module for handling system logs from Okta.


## okta [_okta]

Fields from Okta.

**`okta.uuid`**
:   The unique identifier of the Okta LogEvent.

type: keyword


**`okta.event_type`**
:   The type of the LogEvent.

type: keyword


**`okta.version`**
:   The version of the LogEvent.

type: keyword


**`okta.severity`**
:   The severity of the LogEvent. Must be one of DEBUG, INFO, WARN, or ERROR.

type: keyword


**`okta.display_message`**
:   The display message of the LogEvent.

type: keyword



## actor [_actor]

Fields that let you store information of the actor for the LogEvent.

**`okta.actor.id`**
:   Identifier of the actor.

type: keyword


**`okta.actor.type`**
:   Type of the actor.

type: keyword


**`okta.actor.alternate_id`**
:   Alternate identifier of the actor.

type: keyword


**`okta.actor.display_name`**
:   Display name of the actor.

type: keyword



## client [_client_4]

Fields that let you store information about the client of the actor.

**`okta.client.ip`**
:   The IP address of the client.

type: ip



## user_agent [_user_agent_2]

Fields about the user agent information of the client.

**`okta.client.user_agent.raw_user_agent`**
:   The raw informaton of the user agent.

type: keyword


**`okta.client.user_agent.os`**
:   The OS informaton.

type: keyword


**`okta.client.user_agent.browser`**
:   The browser informaton of the client.

type: keyword


**`okta.client.zone`**
:   The zone information of the client.

type: keyword


**`okta.client.device`**
:   The information of the client device.

type: keyword


**`okta.client.id`**
:   The identifier of the client.

type: keyword



## outcome [_outcome]

Fields that let you store information about the outcome.

**`okta.outcome.reason`**
:   The reason of the outcome.

type: keyword


**`okta.outcome.result`**
:   The result of the outcome. Must be one of: SUCCESS, FAILURE, SKIPPED, ALLOW, DENY, CHALLENGE, UNKNOWN.

type: keyword


**`okta.target`**
:   The list of targets.

type: flattened



## transaction [_transaction]

Fields that let you store information about related transaction.

**`okta.transaction.id`**
:   Identifier of the transaction.

type: keyword


**`okta.transaction.type`**
:   The type of transaction. Must be one of "WEB", "JOB".

type: keyword



## debug_context [_debug_context]

Fields that let you store information about the debug context.


## debug_data [_debug_data]

The debug data.

**`okta.debug_context.debug_data.device_fingerprint`**
:   The fingerprint of the device.

type: keyword


**`okta.debug_context.debug_data.factor`**
:   The factor used for authentication.

type: keyword


**`okta.debug_context.debug_data.request_id`**
:   The identifier of the request.

type: keyword


**`okta.debug_context.debug_data.request_uri`**
:   The request URI.

type: keyword


**`okta.debug_context.debug_data.threat_suspected`**
:   Threat suspected.

type: keyword


**`okta.debug_context.debug_data.risk_behaviors`**
:   The set of behaviors that contribute to a risk assessment.

type: keyword


**`okta.debug_context.debug_data.risk_level`**
:   The risk level assigned to the sign in attempt.

type: keyword


**`okta.debug_context.debug_data.risk_reasons`**
:   The reasons for the risk.

type: keyword


**`okta.debug_context.debug_data.url`**
:   The URL.

type: keyword


**`okta.debug_context.debug_data.flattened`**
:   The complete debug_data object.

type: flattened



## suspicious_activity [_suspicious_activity]

The suspicious activity fields from the debug data.

**`okta.debug_context.debug_data.suspicious_activity.browser`**
:   The browser used.

type: keyword


**`okta.debug_context.debug_data.suspicious_activity.event_city`**
:   The city where the suspicious activity took place.

type: keyword


**`okta.debug_context.debug_data.suspicious_activity.event_country`**
:   The country where the suspicious activity took place.

type: keyword


**`okta.debug_context.debug_data.suspicious_activity.event_id`**
:   The event ID.

type: keyword


**`okta.debug_context.debug_data.suspicious_activity.event_ip`**
:   The IP of the suspicious event.

type: ip


**`okta.debug_context.debug_data.suspicious_activity.event_latitude`**
:   The latitude where the suspicious activity took place.

type: float


**`okta.debug_context.debug_data.suspicious_activity.event_longitude`**
:   The longitude where the suspicious activity took place.

type: float


**`okta.debug_context.debug_data.suspicious_activity.event_state`**
:   The state where the suspicious activity took place.

type: keyword


**`okta.debug_context.debug_data.suspicious_activity.event_transaction_id`**
:   The event transaction ID.

type: keyword


**`okta.debug_context.debug_data.suspicious_activity.event_type`**
:   The event type.

type: keyword


**`okta.debug_context.debug_data.suspicious_activity.os`**
:   The OS of the system from where the suspicious activity occured.

type: keyword


**`okta.debug_context.debug_data.suspicious_activity.timestamp`**
:   The timestamp of when the activity occurred.

type: date



## authentication_context [_authentication_context]

Fields that let you store information about authentication context.

**`okta.authentication_context.authentication_provider`**
:   The information about the authentication provider. Must be one of OKTA_AUTHENTICATION_PROVIDER, ACTIVE_DIRECTORY, LDAP, FEDERATION, SOCIAL, FACTOR_PROVIDER.

type: keyword


**`okta.authentication_context.authentication_step`**
:   The authentication step.

type: integer


**`okta.authentication_context.credential_provider`**
:   The information about credential provider. Must be one of OKTA_CREDENTIAL_PROVIDER, RSA, SYMANTEC, GOOGLE, DUO, YUBIKEY.

type: keyword


**`okta.authentication_context.credential_type`**
:   The information about credential type. Must be one of OTP, SMS, PASSWORD, ASSERTION, IWA, EMAIL, OAUTH2, JWT, CERTIFICATE, PRE_SHARED_SYMMETRIC_KEY, OKTA_CLIENT_SESSION, DEVICE_UDID.

type: keyword


**`okta.authentication_context.issuer`**
:   The information about the issuer.

type: array


**`okta.authentication_context.external_session_id`**
:   The session identifer of the external session if any.

type: keyword


**`okta.authentication_context.interface`**
:   The interface used. e.g., Outlook, Office365, wsTrust

type: keyword



## security_context [_security_context]

Fields that let you store information about security context.


## as [_as_2]

The autonomous system.

**`okta.security_context.as.number`**
:   The AS number.

type: integer



## organization [_organization_2]

The organization that owns the AS number.

**`okta.security_context.as.organization.name`**
:   The organization name.

type: keyword


**`okta.security_context.isp`**
:   The Internet Service Provider.

type: keyword


**`okta.security_context.domain`**
:   The domain name.

type: keyword


**`okta.security_context.is_proxy`**
:   Whether it is a proxy or not.

type: boolean



## request [_request_3]

Fields that let you store information about the request, in the form of list of ip_chain.

**`okta.request.ip_chain`**
:   List of ip_chain objects.

type: flattened


