---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/_set_up_the_oauth_app_in_the_salesforce_2.html
---

# Set up the OAuth App in the Salesforce [_set_up_the_oauth_app_in_the_salesforce_2]

In order to use this integration, users need to create a new Salesforce Application using OAuth. Follow the steps below to create a connected application in Salesforce:

1. Login to [Salesforce](https://login.salesforce.com/) with the same user credentials that the user wants to collect data with.
2. Click on Setup on the top right menu bar. On the Setup page, search for `App Manager` in the `Search Setup` search box at the top of the page, then select `App Manager`.
3. Click *New Connected App*.
4. Provide a name for the connected application. This will be displayed in the App Manager and on its App Launcher tile.
5. Enter the API name. The default is a version of the name without spaces. Only letters, numbers, and underscores are allowed. If the original app name contains any other characters, edit the default name.
6. Enter the contact email for Salesforce.
7. Under the API (Enable OAuth Settings) section of the page, select *Enable OAuth Settings*.
8. In the Callback URL, enter the Instance URL (Please refer to `Salesforce Instance URL`).
9. Select the following OAuth scopes to apply to the connected app:

    * Manage user data via APIs (api).
    * Perform requests at any time (refresh_token, offline_access).
    * (Optional) In case of data collection, if any permission issues arise, add the Full access (full) scope.

10. Select *Require Secret for the Web Server Flow* to require the app’s client secret in exchange for an access token.
11. Select *Require Secret for Refresh Token Flow* to require the app’s client secret in the authorization request of a refresh token and hybrid refresh token flow.
12. Click Save. It may take approximately 10 minutes for the changes to take effect.
13. Click Continue and then under API details, click Manage Consumer Details. Verify the user account using the Verification Code.
14. Copy `Consumer Key` and `Consumer Secret` from the Consumer Details section, which should be populated as values for Client ID and Client Secret respectively in the configuration.

For more details on how to create a Connected App, refer to the Salesforce documentation [here](https://help.salesforce.com/apex/HTViewHelpDoc?id=connected_app_create.htm).

::::{note}
**Enabling real-time events**

To get started with [real-time](https://developer.salesforce.com/blogs/2020/05/introduction-to-real-time-event-monitoring) events, head to setup and into the quick find search for *Event Manager*. Enterprise and Unlimited environments have access to the Logout Event by default, but the remainder of the events need licensing to access [Shield Event Monitoring](https://help.salesforce.com/s/articleView?id=sf.salesforce_shield.htm&type=5).

::::


::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Configure the module [configuring-salesforce-module]

You can further refine the behavior of the `salesforce` module by specifying [variable settings](#salesforce-settings) in the `modules.d/salesforce.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**


### Variable settings [salesforce-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `salesforce` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `salesforce.login.var.paths` instead of `login.var.paths`.
::::



## Fileset settings [_fileset_settings]


### `login` fileset [_login_fileset]

Example config:

```yaml
- module: salesforce
  login:
    enabled: true
    var.initial_interval: 1d
    var.api_version: 56

    var.authentication:
      jwt_bearer_flow:
        enabled: false
        client.id: "my-client-id"
        client.username: "my.email@here.com"
        client.key_path: client_key.pem
        url: https://login.salesforce.com
      user_password_flow:
        enabled: true
        client.id: "my-client-id"
        client.secret: "my-client-secret"
        token_url: "https://login.salesforce.com"
        username: "my.email@here.com"
        password: "password"

    var.url: "https://instance-url.salesforce.com"

    var.event_log_file: true
    var.elf_interval: 1h
    var.log_file_interval: Hourly

    var.real_time: true
    var.real_time_interval: 5m
```

**`var.initial_interval`**
:   The time window for collecting historical data when the input starts. Expects a duration string (e.g. 12h or 7d).

**`var.api_version`**
:   The API version of the Salesforce instance.

**`var.authentication`**
:   Authentication config for connecting to Salesforce API. Supports JWT or user-password auth flows.

**`var.authentication.jwt_bearer_flow.enabled`**
:   Set to true to use JWT authentication.

**`var.authentication.jwt_bearer_flow.client.id`**
:   The client ID for JWT authentication.

**`var.authentication.jwt_bearer_flow.client.username`**
:   The username for JWT authentication.

**`var.authentication.jwt_bearer_flow.client.key_path`**
:   Path to the client key file for JWT authentication.

**`var.authentication.jwt_bearer_flow.url`**
:   The audience URL for JWT authentication.

**`var.authentication.user_password_flow.enabled`**
:   Set to true to use user-password authentication.

**`var.authentication.user_password_flow.client.id`**
:   The client ID for user-password authentication.

**`var.authentication.user_password_flow.client.secret`**
:   The client secret for user-password authentication.

**`var.authentication.user_password_flow.token_url`**
:   The Salesforce token URL for user-password authentication.

**`var.authentication.user_password_flow.username`**
:   The Salesforce username for authentication.

**`var.authentication.user_password_flow.password`**
:   The password for the Salesforce user.

**`var.url`**
:   The URL of the Salesforce instance.

**`var.event_log_file`**
:   Set to true to collect logs from EventLogFile (historical data).

**`var.elf_interval`**
:   Interval for collecting EventLogFile logs, e.g. 1h or 5m.

**`var.log_file_interval`**
:   Either "Hourly" or "Daily". The time interval of each log file from EventLogFile.

**`var.real_time`**
:   Set to true to collect real-time data collection.

**`var.real_time_interval`**
:   Interval for collecting real-time logs, e.g. 30s or 5m.


### `logout` fileset [_logout_fileset]

Example config:

```yaml
- module: salesforce
  logout:
    enabled: true
    var.initial_interval: 1d
    var.api_version: 56

    var.authentication:
      jwt_bearer_flow:
        enabled: false
        client.id: "my-client-id"
        client.username: "my.email@here.com"
        client.key_path: client_key.pem
        url: https://login.salesforce.com
      user_password_flow:
        enabled: true
        client.id: "my-client-id"
        client.secret: "my-client-secret"
        token_url: "https://login.salesforce.com"
        username: "my.email@here.com"
        password: "password"

    var.url: "https://instance-url.salesforce.com"

    var.event_log_file: true
    var.elf_interval: 1h
    var.log_file_interval: Hourly

    var.real_time: true
    var.real_time_interval: 5m
```

**`var.initial_interval`**
:   The time window for collecting historical data when the input starts. Expects a duration string (e.g. 12h or 7d).

**`var.api_version`**
:   The API version of the Salesforce instance.

**`var.authentication`**
:   Authentication config for connecting to Salesforce API. Supports JWT or user-password auth flows.

**`var.authentication.jwt_bearer_flow.enabled`**
:   Set to true to use JWT authentication.

**`var.authentication.jwt_bearer_flow.client.id`**
:   The client ID for JWT authentication.

**`var.authentication.jwt_bearer_flow.client.username`**
:   The username for JWT authentication.

**`var.authentication.jwt_bearer_flow.client.key_path`**
:   Path to the client key file for JWT authentication.

**`var.authentication.jwt_bearer_flow.url`**
:   The audience URL for JWT authentication.

**`var.authentication.user_password_flow.enabled`**
:   Set to true to use user-password authentication.

**`var.authentication.user_password_flow.client.id`**
:   The client ID for user-password authentication.

**`var.authentication.user_password_flow.client.secret`**
:   The client secret for user-password authentication.

**`var.authentication.user_password_flow.token_url`**
:   The Salesforce token URL for user-password authentication.

**`var.authentication.user_password_flow.username`**
:   The Salesforce username for authentication.

**`var.authentication.user_password_flow.password`**
:   The password for the Salesforce user.

**`var.url`**
:   The URL of the Salesforce instance.

**`var.event_log_file`**
:   Set to true to collect logs from EventLogFile (historical data).

**`var.elf_interval`**
:   Interval for collecting EventLogFile logs, e.g. 1h or 5m.

**`var.log_file_interval`**
:   Either "Hourly" or "Daily". The time interval of each log file from EventLogFile.

**`var.real_time`**
:   Set to true to collect real-time data collection.

**`var.real_time_interval`**
:   Interval for collecting real-time logs, e.g. 30s or 5m.


### `setupaudittrail` fileset [_setupaudittrail_fileset]

Example config:

```yaml
- module: salesforce
  setupaudittrail:
    enabled: true
    var.initial_interval: 1d
    var.api_version: 56

    var.authentication:
      jwt_bearer_flow:
        enabled: false
        client.id: "my-client-id"
        client.username: "my.email@here.com"
        client.key_path: client_key.pem
        url: https://login.salesforce.com
      user_password_flow:
        enabled: true
        client.id: "my-client-id"
        client.secret: "my-client-secret"
        token_url: "https://login.salesforce.com"
        username: "my.email@here.com"
        password: "password"

    var.url: "https://instance-url.salesforce.com"

    var.real_time: true
    var.real_time_interval: 5m
```

**`var.initial_interval`**
:   The time window for collecting historical data when the input starts. Expects a duration string (e.g. 12h or 7d).

**`var.api_version`**
:   The API version of the Salesforce instance.

**`var.authentication`**
:   Authentication config for connecting to Salesforce API. Supports JWT or user-password auth flows.

**`var.authentication.jwt_bearer_flow.enabled`**
:   Set to true to use JWT authentication.

**`var.authentication.jwt_bearer_flow.client.id`**
:   The client ID for JWT authentication.

**`var.authentication.jwt_bearer_flow.client.username`**
:   The username for JWT authentication.

**`var.authentication.jwt_bearer_flow.client.key_path`**
:   Path to the client key file for JWT authentication.

**`var.authentication.jwt_bearer_flow.url`**
:   The audience URL for JWT authentication.

**`var.authentication.user_password_flow.enabled`**
:   Set to true to use user-password authentication.

**`var.authentication.user_password_flow.client.id`**
:   The client ID for user-password authentication.

**`var.authentication.user_password_flow.client.secret`**
:   The client secret for user-password authentication.

**`var.authentication.user_password_flow.token_url`**
:   The Salesforce token URL for user-password authentication.

**`var.authentication.user_password_flow.username`**
:   The Salesforce username for authentication.

**`var.authentication.user_password_flow.password`**
:   The password for the Salesforce user.

**`var.url`**
:   The URL of the Salesforce instance.

**`var.real_time`**
:   Set to true to collect real-time data collection.

**`var.real_time_interval`**
:   Interval for collecting real-time logs, e.g. 30s or 5m.


### `apex` fileset [_apex_fileset]

Example config:

```yaml
- module: salesforce
  apex:
    enabled: true
    var.initial_interval: 1d
    var.log_file_interval: Hourly
    var.api_version: 56

    var.authentication:
      jwt_bearer_flow:
        enabled: false
        client.id: "my-client-id"
        client.username: "my.email@here.com"
        client.key_path: client_key.pem
        url: https://login.salesforce.com
      user_password_flow:
        enabled: true
        client.id: "my-client-id"
        client.secret: "my-client-secret"
        token_url: "https://login.salesforce.com"
        username: "my.email@here.com"
        password: "password"

    var.url: "https://instance-url.salesforce.com"

    var.event_log_file: true
    var.elf_interval: 1h
    var.log_file_interval: Hourly
```

**`var.initial_interval`**
:   The time window for collecting historical data when the input starts. Expects a duration string (e.g. 12h or 7d).

**`var.api_version`**
:   The API version of the Salesforce instance.

**`var.authentication`**
:   Authentication config for connecting to Salesforce API. Supports JWT or user-password auth flows.

**`var.authentication.jwt_bearer_flow.enabled`**
:   Set to true to use JWT authentication.

**`var.authentication.jwt_bearer_flow.client.id`**
:   The client ID for JWT authentication.

**`var.authentication.jwt_bearer_flow.client.username`**
:   The username for JWT authentication.

**`var.authentication.jwt_bearer_flow.client.key_path`**
:   Path to the client key file for JWT authentication.

**`var.authentication.jwt_bearer_flow.url`**
:   The audience URL for JWT authentication.

**`var.authentication.user_password_flow.enabled`**
:   Set to true to use user-password authentication.

**`var.authentication.user_password_flow.client.id`**
:   The client ID for user-password authentication.

**`var.authentication.user_password_flow.client.secret`**
:   The client secret for user-password authentication.

**`var.authentication.user_password_flow.token_url`**
:   The Salesforce token URL for user-password authentication.

**`var.authentication.user_password_flow.username`**
:   The Salesforce username for authentication.

**`var.authentication.user_password_flow.password`**
:   The password for the Salesforce user.

**`var.url`**
:   The URL of the Salesforce instance.

**`var.event_log_file`**
:   Set to true to collect logs from EventLogFile (historical data).

**`var.elf_interval`**
:   Interval for collecting EventLogFile logs, e.g. 1h or 5m.

**`var.log_file_interval`**
:   Either "Hourly" or "Daily". The time interval of each log file from EventLogFile.


## Troubleshooting [_troubleshooting]

Here are some common issues and how to resolve them:

**Hitting Salesforce API limits**
:   Reduce the values of `var.real_time_interval` and `var.elf_interval` to poll the API less frequently. Monitor the API usage in your Salesforce instance.

**Connectivity issues**
:   Verify the `var.url` is correct. Check that the user credentials are valid and have the necessary permissions. Ensure network connectivity between the Elastic Agent and Salesforce instance.

**Not seeing any data**
:   Check the Elastic Agent logs for errors. Verify the module configuration is correct, the filesets are enabled, and the intervals are reasonable. Confirm there is log activity in Salesforce for the log types being collected.


## Fields [_fields_47]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-salesforce.md) section.
