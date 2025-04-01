---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/running-on-cloudfoundry.html
---

# Run Metricbeat on Cloud Foundry [running-on-cloudfoundry]

You can use Metricbeat on Cloud Foundry to retrieve and ship metrics.

% However, version {{stack-version}} of Metricbeat has not yet been released, no build is currently available for this version.

## Create Cloud Foundry credentials [_create_cloud_foundry_credentials]

To connect to loggregator and receive the logs, Metricbeat requires credentials created with UAA. The `uaac` command creates the required credentials for connecting to loggregator.

```sh
uaac client add metricbeat --name metricbeat --secret changeme --authorized_grant_types client_credentials,refresh_token --authorities doppler.firehose,cloud_controller.admin_read_only
```

::::{warning}
**Use a unique secret:** The `uaac` command shown here is an example. Remember to replace `changeme` with your secret, and update the `metricbeat.yml` file to use your chosen secret.

::::



## Download Cloud Foundry deploy manifests [_download_cloud_foundry_deploy_manifests]

You deploy Metricbeat as an application with no route.

Cloud Foundry requires that 3 files exist inside of a directory to allow Metricbeat to be pushed. The commands below provide the basic steps for getting it up and running.

```sh subs=true
curl -L -O https://artifacts.elastic.co/downloads/beats/metricbeat/metricbeat-{{stack-version}}-linux-x86_64.tar.gz
tar xzvf metricbeat-{{stack-version}}-linux-x86_64.tar.gz
cd metricbeat-{{stack-version}}-linux-x86_64
curl -L -O https://raw.githubusercontent.com/elastic/beats/master/deploy/cloudfoundry/metricbeat/metricbeat.yml
curl -L -O https://raw.githubusercontent.com/elastic/beats/master/deploy/cloudfoundry/metricbeat/manifest.yml
```

You need to modify the `metricbeat.yml` file to set the `api_address`, `client_id` and `client_secret`.


## Load {{kib}} dashboards [_load_kib_dashboards_2]

Metricbeat comes packaged with various pre-built {{kib}} dashboards that you can use to visualize data in {{kib}}.

If these dashboards are not already loaded into {{kib}}, you must run the Metricbeat `setup` command. To learn how, see [Load {{kib}} dashboards](/reference/metricbeat/load-kibana-dashboards.md).

::::{important}
If you are using a different output other than {{es}}, such as {{ls}}, you need to [Load the index template manually](/reference/metricbeat/metricbeat-template.md#load-template-manually) and [*Load {{kib}} dashboards*](/reference/metricbeat/load-kibana-dashboards.md).

::::



## Deploy Metricbeat [_deploy_metricbeat]

To deploy Metricbeat to Cloud Foundry, run:

```sh
cf push
```

To check the status, run:

```sh
$ cf apps

name       requested state   instances   memory   disk   urls
metricbeat   started           1/1         512M     1G
```

Metric events should start flowing to Elasticsearch. The events are annotated with metadata added by the [add_cloudfoundry_metadata](/reference/metricbeat/add-cloudfoundry-metadata.md) processor.


## Scale Metricbeat [_scale_metricbeat]

A single instance of Metricbeat can ship more than a hundred thousand events per minute. If your Cloud Foundry deployment is producing more events than Metricbeat can collect and ship, the Firehose will start dropping events, and it will mark Metricbeat as a slow consumer. If the problems persist, Metricbeat may be disconnected from the Firehose. In such cases, you will need to scale Metricbeat to avoid losing events.

The main settings you need to take into account are:

* The `shard_id` specified in the [`cloudfoundry` module](/reference/metricbeat/metricbeat-module-cloudfoundry.md). The Firehose will divide the events amongst all the Metricbeat instances with the same value for this setting. All instances with the same `shard_id` should have the same configuration.
* Number of Metricbeat instances. When Metricbeat is deployed as a Cloud Foundry application, it can be scaled up and down like any other application, with `cf scale` or by specifying the number of instances in the manifest.
* [Output configuration](/reference/metricbeat/configuring-output.md). In some cases, you can fine-tune the output configuration to improve the events throughput. Some outputs support multiple workers. The number of workers can be changed to take better advantage of the available resources.

Some basic recommendations to adjust these settings when Metricbeat is not able to collect all events:

* If Metricbeat is hitting its CPU limits, you will need to increase the number of Metricbeat instances deployed with the same `shard_id`.
* If Metricbeat has some spare CPU, there may be some backpressure from the output. Try to increase the number of workers in the output. If this doesn’t help, the bottleneck may be in the network or in the service receiving the events sent by Metricbeat.
* If you need to modify the memory limit of Metricbeat, remember that CPU shares assigned to Cloud Foundry applications depend on the configured memory limit. You may need to check the other recommendations after that.


