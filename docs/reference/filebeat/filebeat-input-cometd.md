---
navigation_title: "CometD"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-cometd.html
---

# CometD input [filebeat-input-cometd]


Use the `https://docs.cometd.org/[cometd]` input to stream the real-time events from a [Salesforce generic subscription Push Topic](https://resources.docs.salesforce.com/sfdc/pdf/api_streaming.pdf).

This input can, for example, be used to receive Login and Logout events that are generated when users log in or out of the Salesforce instance.

Example configuration:

```yaml
filebeat.inputs:
- type: cometd
  channel_name: /event/MyEventStream
  auth.oauth2:
    client.id: my-client-id
    client.secret: my-client-secret
    token_url: https://login.salesforce.com/services/oauth2/token
    user: my.email@mail.com
    password: my-password
```

## Configuration options [_configuration_options_5]

The `cometd` input supports the following configuration options.


### `channel_name` [_channel_name]

Salesforce generic subscription Push Topic name. Required.


### `auth.oauth2.client.id` [_auth_oauth2_client_id_2]

The client ID used as part of the authentication flow. Required.


### `auth.oauth2.client.secret` [_auth_oauth2_client_secret_2]

The client secret used as part of the authentication flow. Required.


### `auth.oauth2.token_url` [_auth_oauth2_token_url_2]

The endpoint that will be used to generate the token during the oauth2 flow. Required.


### `auth.oauth2.user` [_auth_oauth2_user_2]

The user used as part of the authentication flow. It is required for authentication - grant type password. Required.


### `auth.oauth2.password` [_auth_oauth2_password_2]

The password used as part of the authentication flow. It is required for authentication - grant type password. Required.


