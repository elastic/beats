# Development Workflow

In all examples below, `filebeat.dev.yml` is a local development only configuration with VT API key, creds to Elastic Cloud instance, etc.

My `filebeat.dev.yml` looks like this, with appropriate substitutions.

```yml
filebeat.modules:
  - module: virustotal
    livehunt:
      enabled: True
      var.input: httpjson   # httpjson or kafka

      # If consuming events from VirusTotal httpjson
      var.api_key: INSERT-YOUR-VT-API-KEY
      var.vt_fileinfo_url: https://www.virustotal.com/gui/file/
      var.limit: 40 # maximum 40, default is 10

      # If consuming raw events from Kafka
      var.kafka_brokers:
        - 127.0.0.1:9093
      var.kafka_topics:
         - virustotal.raw

cloud.id: "INSERT-YOUR-CLOUD-ID"
cloud.auth: "INSERT-YOUR-CLOUD-AUTH"
```

## Setup

Run filebeat setup to establish ILM policies, Elasticsearch mapping patterns, Kibana index patterns. This needs to happen every time you change a `fields.yml`. This configuration will overwrite existing items, except maybe Kibana index pattern. I just manually delete that before I run setup. I also delete the existing `filebeat-*` index in Elasticsearch too, to avoid mapping conflicts. See notes on `kafka` and `elastidump` below.

```shell
./filebeat -c filebeat.dev.yml -e setup -E setup.template.overwrite=true -E setup.ilm.overwrite=true
```

## Kafka

I added Kafka as an input type because I think some users will find this useful and it's incredibly useful to replay events from VT for development purposes. I used `docker-compose` to standup a local Kafka cluster for development purposes.

```shell
# Download my compose file
curl -O https://gist.githubusercontent.com/dcode/a79d24624aee11ca713250cc5ba02a22/raw/e519b85bad45b3a2f757fbdc2f9808c94969cf13/docker-compose.yml

# Bring up cluster
docker-compose up -d
```

Configure filebeat to use this as an input for the module in your `filebeat.dev.yml`, where `virustotal.raw` is the topic name of the unmodified LiveHunt notification file objects.

```yaml
filebeat.modules:
  - module: virustotal
    livehunt:
      enabled: True
      var.input: kafka
      var.kafka_brokers:
        - 127.0.0.1:9093
      var.kafka_topics:
         - virustotal.raw
```

## Replay Events using Kafka

First, save off existing events from the cluster. Do this before you delete the index in the **setup** step above.

```shell
# Install elasticdump, uses npm/node.js
npm install elasticdump -g

# Install kafkacat and jq
brew install kafkacat jq

# Dump the filebeat index
elasticdump --input=https://elastic:password@elasicsearch-endpoint.es.io:9243/filebeat-* \
  --output=$ \
  | gzip > data/filebeat-virustotal.json.gz

# Replay filebeat data into kafka topic (if setup using compose file above)
gzcat data/filebeat-virustotal.json.gz | jq -cr '._source.event.original' | kafkacat -b 127.0.0.1:9093 -P -t virustotal.raw
```

NOTES:

- `https://elastic:password@elasicsearch-endpoint.es.io:9243`: is the HTTPS endpoint as retreived from the Elastic Cloud panel, with the `elastic` username and password prefixed to the server. You can optionally use an HTTP auth ini file with `elasticdump`. See the `--help` output for specifics.
- Elasticdump can do a lot of things. In this scenario, I'm merely compressing it and writing it to a disk. These are the raw JSON documents as stored in Elasticsearch.
- I'm dumping only indices that match the pattern `filebeat-*`. You can make this more or less specific, as desired.
- I'm using `jq` here to output compact (single line) JSON documents as raw strings, which unquotes the field value.
- kafkacat here is connecting to the local broker `-b`, in producer mode `-P`, and writing to the topic `-t` using data from stdin.

If you wanted to validate or otherwise manipulate the raw data, you can use `kafkacat` in consumer mode `-C`. This example shows the first 10 records. You could pipe these to `jq` to format them.

```shell
kafkacat -b 127.0.0.1:9093 -C -t virustotal.raw | head | jq
```

Configure filebeat to use the kafka input as show above, and run it until all events are replayed. After which, you can switch back to `httpjson` as the input type and stream new data.
