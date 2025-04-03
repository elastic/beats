---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/running-on-cloudfoundry.html
---

# Run Filebeat on Cloud Foundry [running-on-cloudfoundry]

You can use Filebeat on Cloud Foundry to retrieve and ship logs.

% However, version {{stack-version}} of Filebeat has not yet been released, no build is currently available for this version.

## Create Cloud Foundry credentials [_create_cloud_foundry_credentials]

To connect to loggregator and receive the logs, Filebeat requires credentials created with UAA. The `uaac` command creates the required credentials for connecting to loggregator.

```sh
uaac client add filebeat --name filebeat --secret changeme --authorized_grant_types client_credentials,refresh_token --authorities doppler.firehose,cloud_controller.admin_read_only
```

::::{warning}
**Use a unique secret:** The `uaac` command shown here is an example. Remember to replace `changeme` with your secret, and update the `filebeat.yml` file to use your chosen secret.

::::



## Download Cloud Foundry deploy manifests [_download_cloud_foundry_deploy_manifests]

You deploy Filebeat as an application with no route.

Cloud Foundry requires that 3 files exist inside of a directory to allow Filebeat to be pushed. The commands below provide the basic steps for getting it up and running.

```sh subs=true
curl -L -O https://artifacts.elastic.co/downloads/beats/filebeat/filebeat-{{stack-version}}-linux-x86_64.tar.gz
tar xzvf filebeat-{{stack-version}}-linux-x86_64.tar.gz
cd filebeat-{{stack-version}}-linux-x86_64
curl -L -O https://raw.githubusercontent.com/elastic/beats/master/deploy/cloudfoundry/filebeat/filebeat.yml
curl -L -O https://raw.githubusercontent.com/elastic/beats/master/deploy/cloudfoundry/filebeat/manifest.yml
```

You need to modify the `filebeat.yml` file to set the `api_address`, `client_id` and `client_secret`.


## Load {{kib}} dashboards [_load_kib_dashboards_2]

Filebeat comes packaged with various pre-built {{kib}} dashboards that you can use to visualize data in {{kib}}.

If these dashboards are not already loaded into {{kib}}, you must run the Filebeat `setup` command. To learn how, see [Load {{kib}} dashboards](/reference/filebeat/load-kibana-dashboards.md).

The `setup` command does not load the ingest pipelines used to parse log lines. By default, ingest pipelines are set up automatically the first time you run Filebeat and connect to {{es}}.

::::{important}
If you are using a different output other than {{es}}, such as {{ls}}, you need to:

* [Load the index template manually](/reference/filebeat/filebeat-template.md#load-template-manually)
* [*Load {{kib}} dashboards*](/reference/filebeat/load-kibana-dashboards.md)
* [*Load ingest pipelines*](/reference/filebeat/load-ingest-pipelines.md)

::::



## Deploy Filebeat [_deploy_filebeat]

To deploy Filebeat to Cloud Foundry, run:

```sh
cf push
```

To check the status, run:

```sh
$ cf apps

name       requested state   instances   memory   disk   urls
filebeat   started           1/1         512M     1G
```

Log events should start flowing to Elasticsearch. The events are annotated with metadata added by the [add_cloudfoundry_metadata](/reference/filebeat/add-cloudfoundry-metadata.md) processor.


## Scale Filebeat [_scale_filebeat]

A single instance of Filebeat can ship more than a hundred thousand events per minute. If your Cloud Foundry deployment is producing more events than Filebeat can collect and ship, the Firehose will start dropping events, and it will mark Filebeat as a slow consumer. If the problems persist, Filebeat may be disconnected from the Firehose. In such cases, you will need to scale Filebeat to avoid losing events.

The main settings you need to take into account are:

* The `shard_id` specified in the [`cloudfoundry` input configuration](/reference/filebeat/filebeat-input-cloudfoundry.md). The Firehose will divide the events amongst all the Filebeat instances with the same value for this setting. All the instances with the same `shard_id` should have the same configuration.
* Number of Filebeat instances. When Filebeat is deployed as a Cloud Foundry application, it can be scaled up and down like any other application, with `cf scale` or by specifying the number of instances in the manifest.
* [Output configuration](/reference/filebeat/configuring-output.md). In some cases, you can fine-tune the output configuration to improve the events throughput. Some outputs support multiple workers. The number of workers can be changed to take better advantage of the available resources.

Some basic recommendations to adjust these settings when Filebeat is not able to collect all events:

* If Filebeat is hitting its CPU limits, you will need to increase the number of Filebeat instances deployed with the same `shard_id`.
* If Filebeat has some spare CPU, there may be some backpressure from the output. Try to increase the number of workers in the output. If this doesn’t help, the bottleneck may be in the network or in the service receiving the events sent by Filebeat.
* If you need to modify the memory limit of Filebeat, remember that CPU shares assigned to Cloud Foundry applications depend on the configured memory limit. You may need to check the other recommendations after that.


