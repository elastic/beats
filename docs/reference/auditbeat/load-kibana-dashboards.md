---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/load-kibana-dashboards.html
---

# Load Kibana dashboards [load-kibana-dashboards]

Auditbeat comes packaged with example Kibana dashboards, visualizations, and searches for visualizing Auditbeat data in Kibana. Before you can use the dashboards, you need to create the index pattern, `auditbeat-*`, and load the dashboards into Kibana.

To do this, you can either run the `setup` command (as described here) or [configure dashboard loading](/reference/auditbeat/configuration-dashboards.md) in the `auditbeat.yml` config file. This requires a Kibana endpoint configuration. If you didn’t already configure a Kibana endpoint, see [{{kib}} endpoint](/reference/auditbeat/setup-kibana-endpoint.md).


## Load dashboards [load-dashboards]

Make sure Kibana is running before you perform this step. If you are accessing a secured Kibana instance, make sure you’ve configured credentials as described in the [Quick start: installation and configuration](/reference/auditbeat/auditbeat-installation-configuration.md).

To load the recommended index template for writing to {{es}} and deploy the sample dashboards for visualizing the data in {{kib}}, use the command that works with your system.

:::::::{tab-set}

::::::{tab-item} DEB
```sh
auditbeat setup --dashboards
```
::::::

::::::{tab-item} RPM
```sh
auditbeat setup --dashboards
```
::::::

::::::{tab-item} MacOS
```sh
./auditbeat setup --dashboards
```
::::::

::::::{tab-item} Linux
```sh
./auditbeat setup --dashboards
```
::::::

::::::{tab-item} Docker
```sh subs=true
docker run --rm --net="host" docker.elastic.co/beats/auditbeat:{{stack-version}} setup --dashboards
```
::::::

::::::{tab-item} Windows
Open a PowerShell prompt as an Administrator (right-click the PowerShell icon and select **Run As Administrator**).

From the PowerShell prompt, change to the directory where you installed Auditbeat, and run:

```sh
PS > .\auditbeat.exe setup --dashboards
```
::::::

:::::::
For more options, such as loading customized dashboards, see [Importing Existing Beat Dashboards](../../extend/import-dashboards.md).
If you’ve configured the Logstash output, see [Load dashboards for Logstash output](#load-dashboards-logstash).


## Load dashboards for Logstash output [load-dashboards-logstash]

During dashboard loading, Auditbeat connects to Elasticsearch to check version information. To load dashboards when the Logstash output is enabled, you need to temporarily disable the Logstash output and enable Elasticsearch. To connect to a secured Elasticsearch cluster, you also need to pass Elasticsearch credentials.

::::{tip}
The example shows a hard-coded password, but you should store sensitive values in the [secrets keystore](/reference/auditbeat/keystore.md).
::::


:::::::{tab-set}

::::::{tab-item} DEB
```sh
auditbeat setup -e \
  -E output.logstash.enabled=false \
  -E output.elasticsearch.hosts=['localhost:9200'] \
  -E output.elasticsearch.username=auditbeat_internal \
  -E output.elasticsearch.password=YOUR_PASSWORD \
  -E setup.kibana.host=localhost:5601
```
::::::

::::::{tab-item} RPM
```sh
auditbeat setup -e \
  -E output.logstash.enabled=false \
  -E output.elasticsearch.hosts=['localhost:9200'] \
  -E output.elasticsearch.username=auditbeat_internal \
  -E output.elasticsearch.password=YOUR_PASSWORD \
  -E setup.kibana.host=localhost:5601
```
::::::

::::::{tab-item} MacOS
```sh
./auditbeat setup -e \
  -E output.logstash.enabled=false \
  -E output.elasticsearch.hosts=['localhost:9200'] \
  -E output.elasticsearch.username=auditbeat_internal \
  -E output.elasticsearch.password=YOUR_PASSWORD \
  -E setup.kibana.host=localhost:5601
```
::::::

::::::{tab-item} Linux
```sh
./auditbeat setup -e \
  -E output.logstash.enabled=false \
  -E output.elasticsearch.hosts=['localhost:9200'] \
  -E output.elasticsearch.username=auditbeat_internal \
  -E output.elasticsearch.password=YOUR_PASSWORD \
  -E setup.kibana.host=localhost:5601
```
::::::

::::::{tab-item} Docker
```sh subs=true
docker run --rm --net="host" docker.elastic.co/beats/auditbeat:{{stack-version}} setup -e \
  -E output.logstash.enabled=false \
  -E output.elasticsearch.hosts=['localhost:9200'] \
  -E output.elasticsearch.username=auditbeat_internal \
  -E output.elasticsearch.password=YOUR_PASSWORD \
  -E setup.kibana.host=localhost:5601
```
::::::

::::::{tab-item} Windows
Open a PowerShell prompt as an Administrator (right-click the PowerShell icon and select **Run As Administrator**).

From the PowerShell prompt, change to the directory where you installed Auditbeat, and run:

```sh
PS > .\auditbeat.exe setup -e `
  -E output.logstash.enabled=false `
  -E output.elasticsearch.hosts=['localhost:9200'] `
  -E output.elasticsearch.username=auditbeat_internal `
  -E output.elasticsearch.password=YOUR_PASSWORD `
  -E setup.kibana.host=localhost:5601
```
::::::

:::::::
