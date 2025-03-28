---
navigation_title: "Configuration options"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/loggingplugin/current/log-driver-configuration.html
---

# Elastic Logging Plugin configuration options [log-driver-configuration]


Use the following options to configure the Elastic Logging Plugin for Docker. You can pass these options with the `--log-opt` flag when you start a container, or you can set them in the `daemon.json` file for all containers.

* [{{ecloud}} options](#cloud-options)
* [{{es}} output options](#es-output-options)


## Usage examples [_usage_examples]

To set configuration options when you start a container:

```sh subs=true
docker run --log-driver=elastic/elastic-logging-plugin:{{stack-version}} \
           --log-opt hosts="https://myhost:9200" \
           --log-opt user="myusername" \
           --log-opt password="mypassword" \
           -it debian:jessie /bin/bash
```

To set configuration options for all containers in the `daemon.json` file:

```json subs=true
{
  "log-driver" : "elastic/elastic-logging-plugin:{{stack-version}}",
  "log-opts" : {
    "hosts" : "https://myhost:9200",
    "user" : "myusername",
    "password" : "mypassword"
  }
}
```

For more examples, see [Usage examples](/reference/loggingplugin/log-driver-usage-examples.md).


## {{ecloud}} options [cloud-options]

| Option | Description |
| --- | --- |
| `cloud_id` | The Cloud ID found in the Elastic Cloud web console. This ID isused to resolve the {{stack}} URLs when connecting to {{ess}} on {{ecloud}}. |
| `cloud_auth` | The username and password combination for connecting to {{ess}} on {{ecloud}}. Theformat is `"username:password"`. |


## {{es}} output options [es-output-options]

| Option | Default | Description |
| --- | --- | --- |
| `hosts` | `"localhost:9200"` | The list of {{es}} nodes to connect to. Specify each node as a `URL` or`IP:PORT`. For example: `http://192.0.2.0`, `https://myhost:9230` or`192.0.2.0:9300`. If no port is specified, the default is `9200`. |
| `user` |  | The basic authentication username for connecting to {{es}}. |
| `password` |  | The basic authentication password for connecting to {{es}}. |
| `index` |  | A [format string](/reference/libbeat/config-file-format-type.md#_format_string_sprintf)value that specifies the index to write events to when you’re using dailyindices. For example: `"dockerlogs-%{+yyyy.MM.dd}"`. |
| **Advanced:** |
| `name` | `testbeat` | A custom value that will be inserted into the document as `agent.name`.If not set, it will be the hostname of Docker host. |
| `backoff_init` | `1s` | The number of seconds to wait before trying to reconnect to {{es}} aftera network error. After waiting `backoff.init` seconds, the Elastic Logging Plugintries to reconnect. If the attempt fails, the backoff timer is increasedexponentially up to `backoff.max`. After a successful connection, the backofftimer is reset. |
| `backoff_max` | `60s` | The maximum number of seconds to wait before attempting to connect to{{es}} after a network error. |
| `api_key` |  | Instead of using usernames and passwords,you can use API keys to secure communication with {{es}}. |
| `pipeline` |  | A format string value that specifies the {{es}} ingest pipeline to write events to. |
| `timeout` | `90` | The http request timeout in seconds for the Elasticsearch request. |
| `proxy_url` |  | The URL of the proxy to use when connecting to the Elasticsearch servers. Thevalue may be either a complete URL or a `host[:port]`, in which case the `http`scheme is assumed. If a value is not specified through the configuration filethen proxy environment variables are used. See the[Go documentation](https://golang.org/pkg/net/http/#ProxyFromEnvironment)for more information about the environment variables. |


## Configuring the local log [local-log-opts]

This plugin fully supports `docker logs`, and it maintains a local copy of logs that can be read without a connection to Elasticsearch. The plugin mounts the `/var/lib/docker` directory on the host to write logs to `/var/log/containers` on the host. If you want to change the log location on the host, you must change the mount inside the plugin:

1. Disable the plugin:

    ```sh subs=true
    docker plugin disable elastic/elastic-logging-plugin:{{stack-version}}
    ```

2. Set the bindmount directory:

    ```sh subs=true
    docker plugin set elastic/elastic-logging-plugin:{{stack-version}} LOG_DIR.source=NEW_LOG_LOCATION
    ```

3. Enable the plugin:

    ```sh subs=true
    docker plugin enable elastic/elastic-logging-plugin:{{stack-version}}
    ```


The local log also supports the `max-file`, `max-size` and `compress` options that are [a part of the Docker default file logger](https://docs.docker.com/config/containers/logging/json-file/#options). For example:

```sh subs=true
docker run --log-driver=elastic/elastic-logging-plugin:{{stack-version}} \
           --log-opt hosts="myhost:9200" \
           --log-opt user="myusername" \
           --log-opt password="mypassword" \
           --log-opt max-file=10 \
           --log-opt max-size=5M \
           --log-opt compress=true \
           -it debian:jessie /bin/bash
```

In situations where logs can’t be easily managed, for example, you can also configure the plugin to remove log files when a container is stopped. This will prevent you from reading logs on a stopped container, but it will rotate logs without user intervention. To enable removal of logs for stopped containers, you must change the `DESTROY_LOGS_ON_STOP` environment variable:

1. Disable the plugin:

    ```sh subs=true
    docker plugin disable elastic/elastic-logging-plugin:{{stack-version}}
    ```

2. Enable log removal:

    ```sh subs=true
    docker plugin set elastic/elastic-logging-plugin:{{stack-version}} DESTROY_LOGS_ON_STOP=true
    ```

3. Enable the plugin:

    ```sh subs=true
    docker plugin enable elastic/elastic-logging-plugin:{{stack-version}}
    ```


