---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-salesforce.html
---

# Salesforce fields [exported-fields-salesforce]

Salesforce Module


## salesforce [_salesforce]

Fileset for ingesting Salesforce Apex logs.

**`salesforce.instance_url`**
:   The Instance URL of the Salesforce instance.

type: keyword



## apex [_apex]

Fileset for ingesting Salesforce Apex logs.

**`salesforce.apex.document_id`**
:   Unique ID of the Apex document.

type: keyword


**`salesforce.apex.action`**
:   Action performed by the callout.

type: keyword


**`salesforce.apex.callout_time`**
:   Time spent waiting on web service callouts, in milliseconds.

type: float


**`salesforce.apex.class_name`**
:   The Apex class name. If the class is part of a managed package, this string includes the package namespace.

type: keyword


**`salesforce.apex.client_name`**
:   The name of the client that’s using Salesforce services. This field is an optional parameter that can be passed in API calls. If blank, the caller didn’t specify a client in the CallOptions header.

type: keyword


**`salesforce.apex.cpu_time`**
:   The CPU time in milliseconds used to complete the request.

type: float


**`salesforce.apex.db_blocks`**
:   Indicates how much activity is occurring in the database. A high value for this field suggests that adding indexes or filters on your queries would benefit performance.

type: long


**`salesforce.apex.db_cpu_time`**
:   The CPU time in milliseconds to complete the request. Indicates the amount of activity taking place in the database layer during the request.

type: float


**`salesforce.apex.db_total_time`**
:   Time (in milliseconds) spent waiting for database processing in aggregate for all operations in the request. Compare this field to cpu_time to determine whether performance issues are occurring in the database layer or in your own code.

type: float


**`salesforce.apex.entity`**
:   Name of the external object being accessed.

type: keyword


**`salesforce.apex.entity_name`**
:   The name of the object affected by the trigger.

type: keyword


**`salesforce.apex.entry_point`**
:   The entry point for this Apex execution.

type: keyword


**`salesforce.apex.event_type`**
:   The type of event.

type: keyword


**`salesforce.apex.execute_ms`**
:   How long it took (in milliseconds) for Salesforce to prepare and execute the query. Available in API version 42.0 and later.

type: float


**`salesforce.apex.fetch_ms`**
:   How long it took (in milliseconds) to retrieve the query results from the external system. Available in API version 42.0 and later.

type: float


**`salesforce.apex.filter`**
:   Field expressions to filter which rows to return. Corresponds to WHERE in SOQL queries.

type: keyword


**`salesforce.apex.is_long_running_request`**
:   Indicates whether the request is counted against your org’s concurrent long-running Apex request limit (true) or not (false).

type: keyword


**`salesforce.apex.limit`**
:   Maximum number of rows to return for a query. Corresponds to LIMIT in SOQL queries.

type: long


**`salesforce.apex.limit_usage_pct`**
:   The percentage of Apex SOAP calls that were made against the organization’s limit.

type: float


**`salesforce.apex.login_key`**
:   The string that ties together all events in a given user’s login session. It starts with a login event and ends with either a logout event or the user session expiring.

type: keyword


**`salesforce.apex.media_type`**
:   The media type of the response.

type: keyword


**`salesforce.apex.message`**
:   Error or warning message associated with the failed call.

type: text


**`salesforce.apex.method_name`**
:   The name of the calling Apex method.

type: keyword


**`salesforce.apex.fields_count`**
:   The number of fields or columns, where applicable.

type: long


**`salesforce.apex.soql_queries_count`**
:   The number of SOQL queries that were executed during the event.

type: long


**`salesforce.apex.offset`**
:   Number of rows to skip when paging through a result set. Corresponds to OFFSET in SOQL queries.

type: long


**`salesforce.apex.orderby`**
:   Field or column to use for sorting query results, and whether to sort the results in ascending (default) or descending order. Corresponds to ORDER BY in SOQL queries.

type: keyword


**`salesforce.apex.organization_id`**
:   The 15-character ID of the organization.

type: keyword


**`salesforce.apex.query`**
:   The SOQL query, if one was performed.

type: keyword


**`salesforce.apex.quiddity`**
:   The type of outer execution associated with this event.

type: keyword


**`salesforce.apex.request_id`**
:   The unique ID of a single transaction. A transaction can contain one or more events. Each event in a given transaction has the same request_id.

type: keyword


**`salesforce.apex.request_status`**
:   The status of the request for a page view or user interface action.

type: keyword


**`salesforce.apex.rows_total`**
:   Total number of records in the result set. The value is always -1 if the custom adapter’s DataSource.Provider class doesn’t declare the QUERY_TOTAL_SIZE capability.

type: long


**`salesforce.apex.rows_fetched`**
:   Number of rows fetched by the callout. Available in API version 42.0 and later.

type: long


**`salesforce.apex.rows_processed`**
:   The number of rows that were processed in the request.

type: long


**`salesforce.apex.run_time`**
:   The amount of time that the request took in milliseconds.

type: float


**`salesforce.apex.select`**
:   Comma-separated list of fields being queried. Corresponds to SELECT in SOQL queries.

type: keyword


**`salesforce.apex.subqueries`**
:   Reserved for future use.

type: keyword


**`salesforce.apex.throughput`**
:   Number of records retrieved in one second.

type: float


**`salesforce.apex.trigger_id`**
:   The 15-character ID of the trigger that was fired.

type: keyword


**`salesforce.apex.trigger_name`**
:   For triggers coming from managed packages, trigger_name includes a namespace prefix separated with a . character. If no namespace prefix is present, the trigger is from an unmanaged trigger.

type: keyword


**`salesforce.apex.trigger_type`**
:   The type of this trigger.

type: keyword


**`salesforce.apex.type`**
:   The type of Apex callout.

type: keyword


**`salesforce.apex.uri`**
:   The URI of the page that’s receiving the request.

type: keyword


**`salesforce.apex.uri_derived_id`**
:   The 18-character case-safe ID of the URI of the page that’s receiving the request.

type: keyword


**`salesforce.apex.user_agent`**
:   The numeric code for the type of client used to make the request (for example, the browser, application, or API).

type: keyword


**`salesforce.apex.user_id_derived`**
:   The 18-character case-safe ID of the user who’s using Salesforce services through the UI or the API.

type: keyword



## salesforce.login [_salesforce_login]

Fileset for ingesting Salesforce Login (REST) logs.

**`salesforce.login.document_id`**
:   Unique Id.

type: keyword


**`salesforce.login.application`**
:   The application used to access the organization.

type: keyword


**`salesforce.login.api.type`**
:   The type of Salesforce API request.

type: keyword


**`salesforce.login.api.version`**
:   The version of the Salesforce API that’s being used.

type: keyword


**`salesforce.login.auth.service_id`**
:   The authentication method used by a third-party identification provider for an OpenID Connect single sign-on protocol.

type: keyword


**`salesforce.login.auth.method_reference`**
:   The authentication method used by a third-party identification provider for an OpenID Connect single sign-on protocol. This field is available in API version 51.0 and later.

type: keyword


**`salesforce.login.session.level`**
:   Session-level security controls user access to features that support it, such as connected apps and reporting. This field is available in API version 42.0 and later.

type: text


**`salesforce.login.session.key`**
:   The user’s unique session ID. Use this value to identify all user events within a session. When a user logs out and logs in again, a new session is started. For LoginEvent, this field is often null because the event is captured before a session is created. For example, vMASKIU6AxEr+Op5. This field is available in API version 46.0 and later.

type: keyword


**`salesforce.login.key`**
:   The string that ties together all events in a given user’s login session. It starts with a login event and ends with either a logout event or the user session expiring.

type: keyword


**`salesforce.login.history_id`**
:   Tracks a user session so you can correlate user activity with a particular login instance. This field is also available on the LoginHistory, AuthSession, and other objects, making it easier to trace events back to a user’s original authentication.

type: keyword


**`salesforce.login.type`**
:   The type of login used to access the session.

type: keyword


**`salesforce.login.geo_id`**
:   The Salesforce ID of the LoginGeo object associated with the login user’s IP address.

type: keyword


**`salesforce.login.additional_info`**
:   JSON serialization of additional information that’s captured from the HTTP headers during a login request.

type: text


**`salesforce.login.client_version`**
:   The version number of the login client. If no version number is available, “Unknown” is returned.

type: keyword


**`salesforce.login.client_ip`**
:   The IP address of the client that’s using Salesforce services. A Salesforce internal IP (such as a login from Salesforce Workbench or AppExchange) is shown as “Salesforce.com IP”.

type: keyword


**`salesforce.login.cpu_time`**
:   The CPU time in milliseconds used to complete the request. This field indicates the amount of activity taking place in the app server layer.

type: long


**`salesforce.login.db_time_total`**
:   The time in nanoseconds for a database round trip. Includes time spent in the JDBC driver, network to the database, and DB’s CPU time. Compare this field to cpu_time to determine whether performance issues are occurring in the database layer or in your own code.

type: double


**`salesforce.login.event_type`**
:   The type of event. The value is always Login.

type: keyword


**`salesforce.login.organization_id`**
:   The 15-character ID of the organization.

type: keyword


**`salesforce.login.request_id`**
:   The unique ID of a single transaction. A transaction can contain one or more events. Each event in a given transaction has the same REQUEST_ID.

type: keyword


**`salesforce.login.request_status`**
:   The status of the request for a page view or user interface action.

type: keyword


**`salesforce.login.run_time`**
:   The amount of time that the request took in milliseconds.

type: long


**`salesforce.login.user_id`**
:   The 15-character ID of the user who’s using Salesforce services through the UI or the API.

type: keyword


**`salesforce.login.uri_id_derived`**
:   The 18-character case insensitive ID of the URI of the page that’s receiving the request.

type: keyword


**`salesforce.login.evaluation_time`**
:   The amount of time it took to evaluate the transaction security policy, in milliseconds.

type: float


**`salesforce.login.login_type`**
:   The type of login used to access the session.

type: keyword



## salesforce.logout [_salesforce_logout]

Fileset for parsing Salesforce Logout (REST) logs.

**`salesforce.logout.document_id`**
:   Unique Id.

type: keyword


**`salesforce.logout.session.key`**
:   The user’s unique session ID. You can use this value to identify all user events within a session. When a user logs out and logs in again, a new session is started.

type: keyword


**`salesforce.logout.session.level`**
:   The security level of the session that was used when logging out (e.g. Standard Session or High-Assurance Session).

type: text


**`salesforce.logout.session.type`**
:   The session type that was used when logging out (e.g. API, Oauth2 or UI).

type: keyword


**`salesforce.logout.login_key`**
:   The string that ties together all events in a given user’s login session. It starts with a login event and ends with either a logout event or the user session expiring.

type: keyword


**`salesforce.logout.api.type`**
:   The type of Salesforce API request.

type: keyword


**`salesforce.logout.api.version`**
:   The version of the Salesforce API that’s being used.

type: keyword


**`salesforce.logout.app_type`**
:   The application type that was in use upon logging out.

type: keyword


**`salesforce.logout.browser_type`**
:   The identifier string returned by the browser used at login.

type: keyword


**`salesforce.logout.client_version`**
:   The version of the client that was in use upon logging out.

type: keyword


**`salesforce.logout.event_type`**
:   The type of event. The value is always Logout.

type: keyword


**`salesforce.logout.organization_by_id`**
:   The 15-character ID of the organization.

type: keyword


**`salesforce.logout.platform_type`**
:   The code for the client platform. If a timeout caused the logout, this field is null.

type: keyword


**`salesforce.logout.resolution_type`**
:   The screen resolution of the client. If a timeout caused the logout, this field is null.

type: keyword


**`salesforce.logout.user_id`**
:   The 15-character ID of the user who’s using Salesforce services through the UI or the API.

type: keyword


**`salesforce.logout.user_id_derived`**
:   The 18-character case-safe ID of the user who’s using Salesforce services through the UI or the API.

type: keyword


**`salesforce.logout.user_initiated_logout`**
:   The value is 1 if the user intentionally logged out of the organization by clicking the Logout button. If the user’s session timed out due to inactivity or another implicit logout action, the value is 0.

type: keyword


**`salesforce.logout.created_by_id`**
:   Unavailable

type: keyword


**`salesforce.logout.event_identifier`**
:   This field is populated only when the activity that this event monitors requires extra authentication, such as multi-factor authentication. In this case, Salesforce generates more events and sets the RelatedEventIdentifier field of the new events to the value of the EventIdentifier field of the original event. Use this field with the EventIdentifier field to correlate all the related events. If no extra authentication is required, this field is blank.

type: keyword


**`salesforce.logout.organization_id`**
:   The 15-character ID of the organization.

type: keyword



## salesforce.setup_audit_trail [_salesforce_setup_audit_trail]

Fileset for ingesting Salesforce SetupAuditTrail logs.

**`salesforce.setup_audit_trail.document_id`**
:   Unique Id.

type: keyword


**`salesforce.setup_audit_trail.created_by_context`**
:   The context under which the Setup change was made. For example, if Einstein uses cloud-to-cloud services to make a change in Setup, the value of this field is Einstein.

type: keyword


**`salesforce.setup_audit_trail.created_by_id`**
:   Unknown

type: keyword


**`salesforce.setup_audit_trail.created_by_issuer`**
:   Reserved for future use.

type: keyword


**`salesforce.setup_audit_trail.delegate_user`**
:   The Login-As user who executed the action in Setup. If a Login-As user didn’t perform the action, this field is blank. This field is available in API version 35.0 and later.

type: keyword


**`salesforce.setup_audit_trail.display`**
:   The full description of changes made in Setup. For example, if the Action field has a value of PermSetCreate, the Display field has a value like “Created permission set MAD: with user license Salesforce.

type: keyword


**`salesforce.setup_audit_trail.responsible_namespace_prefix`**
:   Unknown

type: keyword


**`salesforce.setup_audit_trail.section`**
:   The section in the Setup menu where the action occurred. For example, Manage Users or Company Profile.

type: keyword


