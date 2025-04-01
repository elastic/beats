---
navigation_title: "Usage examples"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/loggingplugin/current/log-driver-usage-examples.html
---

# Elastic Logging Plugin usage examples [log-driver-usage-examples]


The following examples show common configurations for the Elastic Logging Plugin.


## Send Docker logs to {{es}} [_send_docker_logs_to_es]

**Docker run command:**

```sh subs=true
docker run --log-driver=elastic/elastic-logging-plugin:{{stack-version}} \
           --log-opt hosts="myhost:9200" \
           --log-opt user="myusername" \
           --log-opt password="mypassword" \
           -it debian:jessie /bin/bash
```

**Daemon configuration:**

```json subs=true
{
  "log-driver" : "elastic/elastic-logging-plugin:{{stack-version}}",
  "log-opts" : {
    "hosts" : "myhost:9200",
    "user" : "myusername",
    "password" : "mypassword",
  }
}
```


## Send Docker logs to {{ess}} on {{ecloud}} [_send_docker_logs_to_ess_on_ecloud]

**Docker run command:**

```sh subs=true
docker run --log-driver=elastic/elastic-logging-plugin:{{stack-version}} \
           --log-opt cloud_id="MyElasticStack:daMbY2VudHJhbDekZ2NwLmN4b3VkLmVzLmliJDVkYmQwtGJiYjs0NTRiN4Q5ODJmNGUwm1IxZmFkNjM5JDFiNjdkMDE4MTgxMTQzNTM5ZGFiYWJjZmY0OWIyYWE5" \
           --log-opt cloud_auth="myusername:mypassword" \
           -it debian:jessie /bin/bash
```

**Daemon configuration:**

```json subs=true
{
  "log-driver" : "elastic/elastic-logging-plugin:{{stack-version}}",
  "log-opts" : {
    "cloud_id" : "MyElasticStack:daMbY2VudHJhbDekZ2NwLmN4b3VkLmVzLmliJDVkYmQwtGJiYjs0NTRiN4Q5ODJmNGUwm1IxZmFkNjM5JDFiNjdkMDE4MTgxMTQzNTM5ZGFiYWJjZmY0OWIyYWE5",
    "cloud_auth" : "myusername:mypassword",
    "output.elasticsearch.index" : "elastic-log-driver-%{+yyyy.MM.dd}"
  }
}
```


## Specify a custom index and template [_specify_a_custom_index_and_template]

**Docker run command:**

```sh subs=true
docker run --log-driver=elastic/elastic-logging-plugin:{{stack-version}} \
           --log-opt hosts="myhost:9200" \
           --log-opt user="myusername" \
           --log-opt password="mypassword" \
           --log-opt index="eld-%{[agent.version]}-%{+yyyy.MM.dd}" \
           -it debian:jessie /bin/bash
```

**Daemon configuration:**

```json subs=true
{
  "log-driver" : "elastic/elastic-logging-plugin:{{stack-version}}",
  "log-opts" : {
    "hosts" : "myhost:9200",
    "user" : "myusername",
    "index" : "eld-%{[agent.version]}-%{+yyyy.MM.dd}",
    "password" : "mypassword",
  }
}
```

