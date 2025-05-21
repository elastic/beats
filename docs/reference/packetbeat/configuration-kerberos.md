---
navigation_title: "Kerberos"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/configuration-kerberos.html
---

# Configure Kerberos [configuration-kerberos]


You can specify Kerberos options with any output or input that supports Kerberos, like {{es}}.

The following encryption types are supported:

* aes128-cts-hmac-sha1-96
* aes128-cts-hmac-sha256-128
* aes256-cts-hmac-sha1-96
* aes256-cts-hmac-sha384-192
* des3-cbc-sha1-kd
* rc4-hmac

Example output config with Kerberos password based authentication:

```yaml
output.elasticsearch.hosts: ["http://my-elasticsearch.elastic.co:9200"]
output.elasticsearch.kerberos.auth_type: password
output.elasticsearch.kerberos.username: "elastic"
output.elasticsearch.kerberos.password: "changeme"
output.elasticsearch.kerberos.config_path: "/etc/krb5.conf"
output.elasticsearch.kerberos.realm: "ELASTIC.CO"
```

The service principal name for the Elasticsearch instance is contructed from these options. Based on this configuration it is going to be `HTTP/my-elasticsearch.elastic.co@ELASTIC.CO`.


## Configuration options [_configuration_options_23]

You can specify the following options in the `kerberos` section of the `packetbeat.yml` config file:


### `enabled` [_enabled_11]

The `enabled` setting can be used to enable the kerberos configuration by setting it to `false`. The default value is `true`.

::::{note}
Kerberos settings are disabled if either `enabled` is set to `false` or the `kerberos` section is missing.
::::



### `auth_type` [_auth_type]

There are two options to authenticate with Kerberos KDC: `password` and `keytab`.

`password` expects the principal name and its password. When choosing `keytab`, you have to specify a principal name and a path to a keytab. The keytab must contain the keys of the selected principal. Otherwise, authentication will fail.


### `config_path` [_config_path]

You need to set the path to the `krb5.conf`, so Packetbeat can find the Kerberos KDC to retrieve a ticket.


### `username` [_username_3]

Name of the principal used to connect to the output.


### `password` [_password_4]

If you configured `password` for `auth_type`, you have to provide a password for the selected principal.


### `keytab` [_keytab]

If you configured `keytab` for `auth_type`, you have to provide the path to the keytab of the selected principal.


### `service_name` [_service_name]

This option can only be configured for Kafka. It is the name of the Kafka service, usually `kafka`.


### `realm` [_realm]

Name of the realm where the output resides.


### `enable_krb5_fast` [_enable_krb5_fast]

Enable Kerberos FAST authentication. This may conflict with some Active Directory installations. The default is `false`.

