---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/load-kibana-dashboards.html
---

# Load Kibana dashboards [load-kibana-dashboards]

::::{tip}
For deeper observability into your infrastructure, you can use the {{metrics-app}} and the {{logs-app}} in {{kib}}. For more details, see [Metrics monitoring](docs-content://solutions/observability/infra-and-hosts/analyze-infrastructure-host-metrics.md) and [Log monitoring](docs-content://solutions/observability/logs/explore-logs.md).
::::


Filebeat comes packaged with example Kibana dashboards, visualizations, and searches for visualizing Filebeat data in Kibana. Before you can use the dashboards, you need to create the index pattern, `filebeat-*`, and load the dashboards into Kibana.

To do this, you can either run the `setup` command (as described here) or [configure dashboard loading](/reference/filebeat/configuration-dashboards.md) in the `filebeat.yml` config file. This requires a Kibana endpoint configuration. If you didn’t already configure a Kibana endpoint, see [{{kib}} endpoint](/reference/filebeat/setup-kibana-endpoint.md).


## Load dashboards [load-dashboards]

Make sure Kibana is running before you perform this step. If you are accessing a secured Kibana instance, make sure you’ve configured credentials as described in the [Quick start: installation and configuration](/reference/filebeat/filebeat-installation-configuration.md).

To load the recommended index template for writing to {{es}} and deploy the sample dashboards for visualizing the data in {{kib}}, use the command that works with your system.

:::::::{tab-set}

::::::{tab-item} DEB
```sh
filebeat setup --dashboards
```
::::::

::::::{tab-item} RPM
```sh
filebeat setup --dashboards
```
::::::

::::::{tab-item} MacOS
```sh
./filebeat setup --dashboards
```
::::::

::::::{tab-item} Linux
```sh
./filebeat setup --dashboards
```
::::::

::::::{tab-item} Docker
```sh subs=true
docker run --rm --net="host" docker.elastic.co/beats/filebeat:{{stack-version}} setup --dashboards
```
::::::

::::::{tab-item} Windows
Open a PowerShell prompt as an Administrator (right-click the PowerShell icon and select **Run As Administrator**).

From the PowerShell prompt, change to the directory where you installed Filebeat, and run:

```sh
PS > .\filebeat.exe setup --dashboards
```
::::::

:::::::
For more options, such as loading customized dashboards, see [Importing Existing Beat Dashboards](http://www.elastic.co/guide/en/beats/devguide/master/import-dashboards.md). If you’ve configured the Logstash output, see [Load dashboards for Logstash output](#load-dashboards-logstash).


## Load dashboards for Logstash output [load-dashboards-logstash]

During dashboard loading, Filebeat connects to Elasticsearch to check version information. To load dashboards when the Logstash output is enabled, you need to temporarily disable the Logstash output and enable Elasticsearch. To connect to a secured Elasticsearch cluster, you also need to pass Elasticsearch credentials.

::::{tip}
The example shows a hard-coded password, but you should store sensitive values in the [secrets keystore](/reference/filebeat/keystore.md).
::::


:::::::{tab-set}

::::::{tab-item} DEB
```sh
filebeat setup -e \
  -E output.logstash.enabled=false \
  -E output.elasticsearch.hosts=['localhost:9200'] \
  -E output.elasticsearch.username=filebeat_internal \
  -E output.elasticsearch.password=YOUR_PASSWORD \
  -E setup.kibana.host=localhost:5601
```
::::::

::::::{tab-item} RPM
```sh
filebeat setup -e \
  -E output.logstash.enabled=false \
  -E output.elasticsearch.hosts=['localhost:9200'] \
  -E output.elasticsearch.username=filebeat_internal \
  -E output.elasticsearch.password=YOUR_PASSWORD \
  -E setup.kibana.host=localhost:5601
```
::::::

::::::{tab-item} MacOS
```sh
./filebeat setup -e \
  -E output.logstash.enabled=false \
  -E output.elasticsearch.hosts=['localhost:9200'] \
  -E output.elasticsearch.username=filebeat_internal \
  -E output.elasticsearch.password=YOUR_PASSWORD \
  -E setup.kibana.host=localhost:5601
```
::::::

::::::{tab-item} Linux
```sh
./filebeat setup -e \
  -E output.logstash.enabled=false \
  -E output.elasticsearch.hosts=['localhost:9200'] \
  -E output.elasticsearch.username=filebeat_internal \
  -E output.elasticsearch.password=YOUR_PASSWORD \
  -E setup.kibana.host=localhost:5601
```
::::::

::::::{tab-item} Docker
```sh subs=true
docker run --rm --net="host" docker.elastic.co/beats/filebeat:{{stack-version}} setup -e \
  -E output.logstash.enabled=false \
  -E output.elasticsearch.hosts=['localhost:9200'] \
  -E output.elasticsearch.username=filebeat_internal \
  -E output.elasticsearch.password=YOUR_PASSWORD \
  -E setup.kibana.host=localhost:5601
```
::::::

::::::{tab-item} Windows
Open a PowerShell prompt as an Administrator (right-click the PowerShell icon and select **Run As Administrator**).

From the PowerShell prompt, change to the directory where you installed Filebeat, and run:

```sh
PS > .\filebeat.exe setup -e `
  -E output.logstash.enabled=false `
  -E output.elasticsearch.hosts=['localhost:9200'] `
  -E output.elasticsearch.username=filebeat_internal `
  -E output.elasticsearch.password=YOUR_PASSWORD `
  -E setup.kibana.host=localhost:5601
```
::::::

:::::::
