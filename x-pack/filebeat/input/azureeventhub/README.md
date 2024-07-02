# azure-eventhub input plugin for Filebeat

## Test Scenarios

Test event:

```json
{
  "records": [
    {
      "ReleaseVersion": "6.2023.14.3+7f34763.release_2023w14_zmoog_5",
      "RoleLocation": "France South",
      "callerIpAddress": "88.14.206.49",
      "category": "Administrative",
      "correlationId": "15e73c11-4990-43fb-abf5-755b4551e501",
      "durationMs": "0",
      "identity": {
        "authorization": {
          "action": "Microsoft.Compute/virtualMachines/deallocate/action",
          "evidence": {
            "principalId": "ee4d999c57f24213adac6192582b8649",
            "principalType": "Group",
            "role": "Owner",
            "roleAssignmentId": "0b47993c5d35401cb0d75a4f00f4728c",
            "roleAssignmentScope": "/subscriptions/12cabcb4-86e8-404f-a3d2-1dc9982f45ca",
            "roleDefinitionId": "8e3af657a8ff443ca75c2fe8c4bcb635"
          },
          "scope": "/subscriptions/12cabcb4-86e8-404f-a3d2-1dc9982f45ca/resourceGroups/tdancheva-integrations/providers/Microsoft.Compute/virtualMachines/azure-host-2"
        },
        "claims": {
          "aio": "AWQAm/8TAAAA6/xwhRYxDjcCZif6YoWZ+QsQMuhT5SHB+ppfzHYY+/sRZ4R2MCnsy1UgKpHzCkrKm/pd3Cou0WkwJE16A5XXl6YXvFdOEYtVvR9Rl1ICI7+s3jIsyqgAt9KnxrUJs7Vk",
          "altsecid": "5::10032002612EEF9A",
          "appid": "c44b4083-3bb0-49c1-b47d-974e53cbdf3c",
          "appidacr": "2",
          "aud": "https://management.core.windows.net/",
          "exp": "1681892540",
          "groups": "6089bd09-85f7-465c-826e-626f83b4b90c,ee4d999c-57f2-4213-adac-6192582b8649",
          "http://schemas.microsoft.com/claims/authnclassreference": "1",
          "http://schemas.microsoft.com/claims/authnmethodsreferences": "pwd,mfa",
          "http://schemas.microsoft.com/identity/claims/identityprovider": "https://sts.windows.net/4fa94b7d-a743-486f-abcc-6c276c44cf4b/",
          "http://schemas.microsoft.com/identity/claims/objectidentifier": "385b609f-6d52-48c6-839c-057d2cd5b1e9",
          "http://schemas.microsoft.com/identity/claims/scope": "user_impersonation",
          "http://schemas.microsoft.com/identity/claims/tenantid": "aa40685b-417d-4664-b4ec-8f7640719adb",
          "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": "tamara.dancheva@elastic.co",
          "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname": "Tamara",
          "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name": "tamara.dancheva@elastic.co",
          "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/nameidentifier": "vvTSrJ-rm3FoWEwZguCZGPOgbhAcYEC0aOWDbdS_w5o",
          "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname": "Dancheva",
          "iat": "1681888229",
          "ipaddr": "88.14.206.49",
          "iss": "https://sts.windows.net/aa40685b-417d-4664-b4ec-8f7640719adb/",
          "name": "Tamara Dancheva",
          "nbf": "1681888229",
          "puid": "1003200290727777",
          "rh": "0.AUgAW2hAqn1BZEa07I92QHGa20ZIf3kAutdPukPawfj2MBNIAOE.",
          "uti": "_xoydzBhcUObV3WTNcBFAA",
          "ver": "1.0",
          "wids": "f2ef992c-3afb-46b9-b7cf-a126ee74c451",
          "xms_tcdt": "1391159646"
        }
      },
      "jobId": "ProxyResourceLongOperationJob:2DGRL:2DTDANCHEVA:3A2DINTEGRATIONS:2DMICROSOFT:3A2ECOMPUTE:3A2FVIRTUALMACHINES:3A|187402E12C07F52B",
      "jobType": "ProxyResourceLongOperationJob",
      "level": "Information",
      "operationName": "MICROSOFT.COMPUTE/VIRTUALMACHINES/DEALLOCATE/ACTION",
      "properties": {
        "entity": "/subscriptions/12cabcb4-86e8-404f-a3d2-1dc9982f45ca/resourceGroups/tdancheva-integrations/providers/Microsoft.Compute/virtualMachines/azure-host-2",
        "eventCategory": "Administrative",
        "hierarchy": "aa40685b-417d-4664-b4ec-8f7640719adb/12cabcb4-86e8-404f-a3d2-1dc9982f45ca",
        "message": "Microsoft.Compute/virtualMachines/deallocate/action"
      },
      "resourceId": "/SUBSCRIPTIONS/12CABCB4-86E8-404F-A3D2-1DC9982F45CA/RESOURCEGROUPS/TDANCHEVA-INTEGRATIONS/PROVIDERS/MICROSOFT.COMPUTE/VIRTUALMACHINES/AZURE-HOST-2",
      "resultSignature": "Succeeded.",
      "resultType": "Success",
      "tenantId": "aa40685b-417d-4664-b4ec-8f7640719adb",
      "time": "2023-06-15:54:46.8676027Z"
    }
  ]
}

```

### Scenario 001: Migration

- Setup
- start with v1
- process 10 events
- check that checkpoint info v1 have been created
- check that the 10 events are processed
- check that checkpoint info v1 have been updated
- stop v1, enable v2, and start v2
- check that checkpoint info v2 have been created
- check that the 10 events are not processed again

#### Setup

- Delete the index `filebeat-8.15.0` from the test cluster.
- Set `storage_account_container` with a new container name.

#### Start with v1

Using the following configuration:

```yaml
# x-pack/filebeat/modules.d/azure.yml

- module: azure
  # All logs
  activitylogs:
    enabled: true
    var:
      eventhub: "eventhubsdkupgrade"
      consumer_group: "$Default"
      connection_string: "<redacted>"
      storage_account: "mbrancageneral"
      storage_account_container: "filebeat-activitylogs-zmoog-0005"
      storage_account_key: "<redacted>"
      storage_account_connection_string: "<redacted>"
      processor_version: "v1"
      migrate_checkpoint: yes
      start_position: "earliest"
```

#### Check that checkpoint info v1 have been created

After the input started successfully, I see four blobs in the
`filebeat-activitylogs-zmoog-0005`, one for each partition.

Here is the content of blob for partition `0`:

```json
{
  "partitionID": "0",
  "epoch": 1,
  "owner": "382ed56f-291c-4801-a70a-13ddbe131040",
  "checkpoint": {
    "offset": "-1",
    "sequenceNumber": 0,
    "enqueueTime": "0001-01-01T00:00:00Z"
  },
  "state": "available",
  "token": "33cdc5d9-7e22-443a-bd6d-197c971967b3"
}
```

All values have their zero value because the input never processed any event.

#### Process 10 events

Use the https://pypi.org/project/eventhubs/ tool to send 10 events to the event hub `eventhubsdkupgrade`:

```shell
export EVENTHUB_CONNECTION_STRING="<redacted>"
export EVENTHUB_NAMESPACE="mbranca-general"
export EVENTHUB_NAME="eventhubsdkupgrade"

$ eh -v eventdata send-batch --lines-from-text-file activitylogs.ndjson --batch-size 40
Sending 10 events to eventhubsdkupgrade
sending batch of 10 events
batch sent successfully
```

The `activitylogs.ndjson` file contains ten copies of the file test event.

#### check that the 10 events are processed

I see the `filebeat-8.15.0` contains 10 documents.

#### check that checkpoint info v1 have been updated

I see the `filebeat-activitylogs-zmoog-0005` container still contains four blobs, but one of them
now has a different size (`235B` instead of `228B`).

The content of blobs `0`, `2`, and `3` is unchanged.

The content of blobs `1` is:

```json
{
  "partitionID": "1",
  "epoch": 1,
  "owner": "382ed56f-291c-4801-a70a-13ddbe131040",
  "checkpoint": {
    "offset": "31680",
    "sequenceNumber": 9,
    "enqueueTime": "2024-06-03T10:34:22.678Z"
  },
  "state": "available",
  "token": "32cd8a2c-a8cf-4f0f-b3cd-9e13c9830beb"
}
```

The `sequenceNumber` changed from `0` to `9`.

#### stop v1, enable v2, and start v2

Stop Filebeat and update the config with the following changes:

```yaml
# x-pack/filebeat/modules.d/azure.yml

- module: azure
  # All logs
  activitylogs:
    enabled: true
    var:
      eventhub: "eventhubsdkupgrade"
      consumer_group: "$Default"
      connection_string: "<redacted>"
      storage_account: "mbrancageneral"
      storage_account_container: "filebeat-activitylogs-zmoog-0005"
      storage_account_key: "<redacted>"
      storage_account_connection_string: "<redacted>" # NOTE: make sure this is set
      processor_version: "v2" # CHANGE: v1 > v2
      migrate_checkpoint: yes
      start_position: "earliest"
```

#### check that checkpoint info v2 have been created

I see we have the following folder:

```text
filebeat-activitylogs-zmoog-0005 / mbranca-general.servicebus.windows.net / eventhubsdkupgrade / $Default / checkpoint
```

The folder containts four blobs `0`, `1`, `2`, and `3` 

The metadata of blobs `0`, `2`, and `3`:

- `offset`: -1
- `sequencenumber`: 0

The metadata of blob `1` is:

- `offset`: 31680
- `sequencenumber`: 9

#### check that the 10 events are not processed again

The index `filebeat-8.15.0` still contains 10 documents, so the input did not reprocessed the same events.

### Scenario 002: ingest 100 events (1 input)

- Setup
- Start v2
- Take a note with the sequencenumber for all partitions
- Process 100 events
- Check that the 100 events are processed
- check that checkpoint info v2 have been updated

#### Setup

- Delete the index `filebeat-8.15.0` from the test cluster.

#### Start v2

Using the following configuration:

```yaml
# x-pack/filebeat/modules.d/azure.yml

- module: azure
  # All logs
  activitylogs:
    enabled: true
    var:
      eventhub: "eventhubsdkupgrade"
      consumer_group: "$Default"
      connection_string: "<redacted>"
      storage_account: "mbrancageneral"
      storage_account_container: "filebeat-activitylogs-zmoog-0005"
      storage_account_key: "<redacted>"
      storage_account_connection_string: "<redacted>"
      processor_version: "v2"
      migrate_checkpoint: yes
      start_position: "earliest"
```

#### Take a note with the sequencenumber for all partitions

Here are the current sequence numbers:

| Partition | Sequence number | Offset |
| --------- | --------------- | ------ |
| 0         | 0               | -1     |
| 1         | 9               | 31680  |
| 2         | 0               | -1     |
| 3         | 0               | -1     |


#### Process 100 events

Edit the `activitylogs.ndjson` to have 100 events.

Send the 100 events:

```shell
$ eh -v eventdata send-batch --lines-from-text-file activitylogs.ndjson --batch-size 40
Sending 100 events to eventhubsdkupgrade
sending batch of 40 events
batch sent successfully
sending batch of 40 events
batch sent successfully
sending batch of 20 events
batch sent successfully
```

#### Check that the 100 events are processed 

I see the `filebeat-8.15.0` contains 100 events.


#### Check that checkpoint info v2 have been updated

Here are the current sequence numbers:

| Partition | Sequence number | Offset |
| --------- | --------------- | ------ |
| 0         | 39              | 137280 |
| 1         | 49              | 172480 |
| 2         | 19              | 66880  |
| 3         | 0               | -1     |

Of the 100 events published, 

- 40 landed on partition 0 (0 > 39)
- 40 landed on partition 1 (9 > 49)
- 20 landed on partition 2 (0 > 19)
- 0 landed on partition 3

Here are the logs:

```shell
$ pbpaste | grep '^{' |  jq -r 'select(."log.logger" == "input.azure-eventhub") | [."@timestamp",."log.level",."log.logger",.message,.partition,.count//0,.acked//0,.error.message//"na",.error] | @tsv' | sort

2024-06-03T12:45:23.791+0200	info	input.azure-eventhub	Input 'azure-eventhub' starting		0	0	na
2024-06-03T12:45:24.379+0200	debug	input.azure-eventhub	blob container already exists, no need to create a new one		0	0	na
2024-06-03T12:45:29.629+0200	info	input.azure-eventhub	checkpoint migration is enabled		0	0	na
2024-06-03T12:45:46.201+0200	info	input.azure-eventhub	event hub information		0	0	na
2024-06-03T12:46:28.779+0200	info	input.azure-eventhub	downloaded checkpoint v1 information for partition		0	0	na
2024-06-03T12:46:35.197+0200	info	input.azure-eventhub	migrating checkpoint v1 information to v2		0	0	na
2024-06-03T12:46:42.561+0200	info	input.azure-eventhub	migrated checkpoint v1 information to v2		0	0	na
2024-06-03T12:46:49.400+0200	info	input.azure-eventhub	downloaded checkpoint v1 information for partition		0	0	na
2024-06-03T12:46:49.400+0200	info	input.azure-eventhub	migrating checkpoint v1 information to v2		0	0	na
2024-06-03T12:46:49.633+0200	info	input.azure-eventhub	migrated checkpoint v1 information to v2		0	0	na
2024-06-03T12:46:49.862+0200	info	input.azure-eventhub	downloaded checkpoint v1 information for partition		0	0	na
2024-06-03T12:46:49.863+0200	info	input.azure-eventhub	migrating checkpoint v1 information to v2		0	0	na
2024-06-03T12:46:50.100+0200	info	input.azure-eventhub	migrated checkpoint v1 information to v2		0	0	na
2024-06-03T12:46:50.333+0200	info	input.azure-eventhub	downloaded checkpoint v1 information for partition		0	0	na
2024-06-03T12:46:50.333+0200	info	input.azure-eventhub	migrating checkpoint v1 information to v2		0	0	na
2024-06-03T12:46:50.567+0200	info	input.azure-eventhub	migrated checkpoint v1 information to v2		0	0	na
2024-06-03T12:46:55.762+0200	info	input.azure-eventhub	starting a partition worker	2	0	0	na
2024-06-03T12:47:08.101+0200	info	input.azure-eventhub	starting a partition worker	3	0	0	na
2024-06-03T12:47:21.621+0200	info	input.azure-eventhub	starting a partition worker	0	0	0	na
2024-06-03T12:47:34.559+0200	info	input.azure-eventhub	starting a partition worker	1	0	0	na
2024-06-03T13:04:44.734+0200	debug	input.azure-eventhub	received events	1	15	0	na
2024-06-03T13:04:45.964+0200	debug	input.azure-eventhub	received events	2	20	0	na
2024-06-03T13:04:46.821+0200	debug	input.azure-eventhub	received events	0	40	0	na
2024-06-03T13:04:49.746+0200	debug	input.azure-eventhub	received events	1	25	0	na
2024-06-03T13:05:00.339+0200	debug	input.azure-eventhub	checkpoint updated	1	0	40	na
2024-06-03T13:05:01.889+0200	debug	input.azure-eventhub	checkpoint updated	0	0	40	na
2024-06-03T13:05:03.167+0200	debug	input.azure-eventhub	checkpoint updated	2	0	20	na
```

### Scenario 003: ingest 100 events (2 inputs)

- Setup
- Start two inputs
- Take a note with the sequencenumber for all partitions
- Process 100 events
- Check that the 100 events are processed
- check that checkpoint info v2 have been updated
- Stop input 2
- Check that input 1 started two new consumer

#### Setup

- Delete the index `filebeat-8.15.0` from the test cluster.


#### Start two inputs

Using the following configuration for all inputs:

```yaml
# x-pack/filebeat/modules.d/azure.yml

- module: azure
  # All logs
  activitylogs:
    enabled: true
    var:
      eventhub: "eventhubsdkupgrade"
      consumer_group: "$Default"
      connection_string: "<redacted>"
      storage_account: "mbrancageneral"
      storage_account_container: "filebeat-activitylogs-zmoog-0005"
      storage_account_key: "<redacted>"
      storage_account_connection_string: "<redacted>"
      processor_version: "v2"
      migrate_checkpoint: yes
      start_position: "earliest"
```

- Started input 1
- Input 1 is running and processing events 

```shell
$ pbpaste | grep '^{' |  jq -r 'select(."log.logger" == "input.azure-eventhub") | [."@timestamp",."log.level",."log.logger",.message,.partition,.count//0,.acked//0,.error.message//"na",.error] | @tsv' | sort

2024-06-03T12:46:55.762+0200	info	input.azure-eventhub	starting a partition worker	2	0	0	na
2024-06-03T12:47:08.101+0200	info	input.azure-eventhub	starting a partition worker	3	0	0	na
2024-06-03T12:47:21.621+0200	info	input.azure-eventhub	starting a partition worker	0	0	0	na
2024-06-03T12:47:34.559+0200	info	input.azure-eventhub	starting a partition worker	1	0	0	na
2024-06-03T13:04:44.734+0200	debug	input.azure-eventhub	received events	1	15	0	na
2024-06-03T13:04:45.964+0200	debug	input.azure-eventhub	received events	2	20	0	na
2024-06-03T13:04:46.821+0200	debug	input.azure-eventhub	received events	0	40	0	na
2024-06-03T13:04:49.746+0200	debug	input.azure-eventhub	received events	1	25	0	na
2024-06-03T13:05:00.339+0200	debug	input.azure-eventhub	checkpoint updated	1	0	40	na
2024-06-03T13:05:01.889+0200	debug	input.azure-eventhub	checkpoint updated	0	0	40	na
2024-06-03T13:05:03.167+0200	debug	input.azure-eventhub	checkpoint updated	2	0	20	na
```

- Started input 2

Input 2 claimed partitions `0` and `3`.

```shell
$ pbpaste | grep '^{' |  jq -r 'select(."log.logger" == "input.azure-eventhub") | [."@timestamp",."log.level",."log.logger",.message,.partition,.count//0,.acked//0,.error.message//"na",.error] | @tsv' | sort

2024-06-03T13:51:33.748+0200	info	input.azure-eventhub	Input 'azure-eventhub' starting		0	0	na
2024-06-03T13:51:37.197+0200	debug	input.azure-eventhub	blob container already exists, no need to create a new one		0	0	na
2024-06-03T13:51:37.197+0200	info	input.azure-eventhub	checkpoint migration is enabled		0	0	na
2024-06-03T13:51:38.986+0200	info	input.azure-eventhub	event hub information		0	0	na
2024-06-03T13:51:39.234+0200	info	input.azure-eventhub	checkpoint v2 information for partition already exists, no migration needed		0	0	na
2024-06-03T13:51:39.234+0200	info	input.azure-eventhub	checkpoint v2 information for partition already exists, no migration needed		0	0	na
2024-06-03T13:51:39.234+0200	info	input.azure-eventhub	checkpoint v2 information for partition already exists, no migration needed		0	0	na
2024-06-03T13:51:39.234+0200	info	input.azure-eventhub	checkpoint v2 information for partition already exists, no migration needed		0	0	na
2024-06-03T13:51:40.728+0200	info	input.azure-eventhub	starting a partition worker	3	0	0	na
2024-06-03T13:52:03.777+0200	info	input.azure-eventhub	starting a partition worker	0	0	0	na
```

Input 1 released partitions `0` and `3`.

```shell
$ pbpaste | grep '^{' |  jq -r 'select(."log.logger" == "input.azure-eventhub") | [."@timestamp",."log.level",."log.logger",.message,.partition,.count//0,.acked//0,.error.message//"na",.error] | @tsv' | sort

2024-06-03T13:51:45.711+0200	debug	input.azure-eventhub	partition resources cleaned up	3	0	0	na
2024-06-03T13:51:45.711+0200	info	input.azure-eventhub	partition worker exited	3	0	0	na
2024-06-03T13:52:08.734+0200	debug	input.azure-eventhub	partition resources cleaned up	0	0	0	na
2024-06-03T13:52:08.734+0200	info	input.azure-eventhub	partition worker exited	0	0	0	na
```

After input 2 started successfully, the two input share 50% of the event hub partitions each:

- input 1: partition 1, 2
- input 2: partition 0, 3

#### Send 100 events

Edit the `activitylogs.ndjson` to have 100 events.

Send the 100 events:

```shell
$ eh -v eventdata send-batch --lines-from-text-file activitylogs.ndjson --batch-size 40
Sending 100 events to eventhubsdkupgrade
sending batch of 40 events
batch sent successfully
sending batch of 40 events
batch sent successfully
sending batch of 20 events
batch sent successfully
```


#### Check that the 100 events are processed 

I see the `filebeat-8.15.0` contains 100 events.


#### Check that checkpoint info v2 have been updated

Here are the current sequence numbers:

Before

| Partition | Sequence number | Offset |
| --------- | --------------- | ------ |
| 0         | 39              | 137280 |
| 1         | 49              | 172480 |
| 2         | 19              | 66880  |
| 3         | 0               | -1     |

After

| Partition | Sequence number | Offset |
| --------- | --------------- | ------ |
| 0         | 59              | 207680 |
| 1         | 49              | 207680 |
| 2         | 59              | 66880  |
| 3         | 39              | 137280 |

Of the 100 events published, 

- 20 landed on partition 0 (39 > 59)
- 0 landed on partition 1 
- 40 landed on partition 2 (19 > 59)
- 40 landed on partition 3 (0 > 39)

The total number of documents increased by 100.

#### Check that documents come from two agents

By running the following query:

```json
POST /index_name/_search
{
  "size": 0,
  "aggs": {
    "agents": {
      "terms": {
        "field": "agent.id.keyword"
      }
    }
  }
}
```

I get this split:

```json
{
  "took": 2,
  "timed_out": false,
  "_shards": {
    "total": 1,
    "successful": 1,
    "skipped": 0,
    "failed": 0
  },
  "hits": {
    "total": {
      "value": 100,
      "relation": "eq"
    },
    "max_score": null,
    "hits": []
  },
  "aggregations": {
    "agents": {
      "doc_count_error_upper_bound": 0,
      "sum_other_doc_count": 0,
      "buckets": [
        {
          "key": "43928b5d-b3c6-4ad9-9a6f-d24d1c3e83bd",
          "doc_count": 60
        },
        {
          "key": "f5f4b7cb-fc0f-4aa2-909f-62fad44d56ff",
          "doc_count": 40
        }
      ]
    }
  }
}
```

#### Stop input 2

Just shut down the input 2.

#### Check that input 1 started two new consumers

After ~10s, the input 1 started two new consumer to claim the partition from input 2:

```shell
2024-06-03T19:25:20.104+0200	info	input.azure-eventhub	starting a partition worker	2	0	0	na
2024-06-03T19:25:32.100+0200	info	input.azure-eventhub	starting a partition worker	1	0	0	na
```

### Scenario 004: Invalid Elasticsearch endpoint

The goal of this scenario is to verify that if the input uses an invalid Elasticsearch endpoint, the input does not update the checkpoint data.

- Setup
- Start one input
- Take a note with the sequencenumber for all partitions
- Send 10 events
- Check that checkpoint info v2 are not updated
- Check that the 10 events are stored in the in-memory queue
- Check that after fixing the problem the input successfully processed the 10 events

#### Setup

- Delete the index `filebeat-8.15.0` from the test cluster.


#### Start one input

Using the following configuration:

```yaml
# x-pack/filebeat/modules.d/azure.yml

- module: azure
  # All logs
  activitylogs:
    enabled: true
    var:
      eventhub: "eventhubsdkupgrade"
      consumer_group: "$Default"
      connection_string: "<redacted>"
      storage_account: "mbrancageneral"
      storage_account_container: "filebeat-activitylogs-zmoog-0005"
      storage_account_key: "<redacted>"
      storage_account_connection_string: "<redacted>"
      processor_version: "v2"
      migrate_checkpoint: yes
      start_position: "earliest"
```

Important: set the `cloud.id` with a deleted deployment, or set `cloud.auth` with invalid credentials. 

```shell
./filebeat -e -v -d * \
    --strict.perms=false \
    --path.home /Users/zmoog/code/projects/elastic/beats/x-pack/filebeat \
    -E cloud.id=<redacted> \
    -E cloud.auth=<redacted> \
    -E gc_percent=100 \
    -E setup.ilm.enabled=false \
    -E setup.template.enabled=false \
    -E output.elasticsearch.allow_older_versions=true
```

The Elasticsearch output must fail to send anything to the cluster.

#### Take a note with the sequencenumber for all partitions

Current checkpoint info are:

| Partition | Sequence number | Offset |
| --------- | --------------- | ------ |
| 0         | 59              | 207680 |
| 1         | 49              | 172480 |
| 2         | 59              | 207680 |
| 3         | 39              | 137280 |

#### Send 10 events

Edit the `activitylogs.ndjson` to have 10 events.

Send the 10 events:

```shell
$ eh -v eventdata send-batch --lines-from-text-file activitylogs.ndjson --batch-size 40

Sending 10 events to eventhubsdkupgrade
sending batch of 10 events
batch sent successfully
```

#### Check that checkpoint info v2 are not updated

The partition `1` received 10 events:

```
2024-06-03T22:55:18.539+0200	debug	input.azure-eventhub	received events	1	10	0	na
```

Current checkpoint info are:

| Partition | Sequence number | Offset |
| --------- | --------------- | ------ |
| 0         | 59              | 207680 |
| 1         | 49              | 172480 |
| 2         | 59              | 207680 |
| 3         | 39              | 137280 |

Partition `1`, and all other partitions checkpoint info as metadata, are unchanged.


#### Check that the 10 events are stored in the in-memory queue

Checking the metrics:

```shell
$ pbpaste | grep "Non-zero" | jq -r '[.["@timestamp"],.component.id,.monitoring.metrics.filebeat.events.active,.monitoring.metrics.libbeat.pipeline.events.active,.monitoring.metrics.libbeat.output.events.total//"n/a",.monitoring.metrics.libbeat.output.events.acked//"n/a",.monitoring.metrics.libbeat.output.events.failed//0] | @tsv' | sort

2024-06-03T22:54:14.956+0200		0	0	n/a	n/a	0
2024-06-03T22:54:44.956+0200		0	0	n/a	n/a	0
2024-06-03T22:55:14.972+0200		0	0	n/a	n/a	0
2024-06-03T22:55:44.958+0200		10	10	n/a	n/a	0
2024-06-03T22:56:14.956+0200		10	10	n/a	n/a	0
2024-06-03T22:56:44.962+0200		10	10	n/a	n/a	0
2024-06-03T22:57:14.957+0200		10	10	n/a	n/a	0
2024-06-03T22:57:44.955+0200		10	10	n/a	n/a	0
2024-06-03T22:58:14.957+0200		10	10	n/a	n/a	0
2024-06-03T22:58:44.956+0200		10	10	n/a	n/a	0
2024-06-03T22:59:14.957+0200		10	10	n/a	n/a	0
2024-06-03T22:59:44.957+0200		10	10	n/a	n/a	0
2024-06-03T23:00:14.957+0200		10	10	n/a	n/a	0
2024-06-03T23:00:44.956+0200		10	10	n/a	n/a	0
2024-06-03T23:01:14.956+0200		10	10	n/a	n/a	0
202e-06-03T23:01:44.955+0200		10	10	n/a	n/a	0
2024-06-03T23:02:14.961+0200		10	10	n/a	n/a	0
2024-06-03T23:02:44.957+0200		10	10	n/a	n/a	0
2024-06-03T23:03:14.955+0200		10	10	n/a	n/a	0
```

I see the `.monitoring.metrics.filebeat.events.active` and `.monitoring.metrics.libbeat.pipeline.events.active` metrics values are both `10`, but `.monitoring.metrics.libbeat.output.events.total` and `.monitoring.metrics.libbeat.output.events.acked` metrics values are both `n/a`.

#### Check that after fixing the problem the input successfully processed the 10 events

- Update `cloud.auth` with valid credentials 
- restart the input

After restarting the input, here are the input metrics:

```shell
$ pbpaste | grep "Non-zero" | jq -r '[.["@timestamp"],.component.id,.monitoring.metrics.filebeat.events.active,.monitoring.metrics.libbeat.pipeline.events.active,.monitoring.metrics.libbeat.output.events.total//"n/a",.monitoring.metrics.libbeat.output.events.acked//"n/a",.monitoring.metrics.libbeat.output.events.failed//0] | @tsv' | sort

2024-06-03T23:25:57.052+0200		0	0	n/a	n/a	0
2024-06-03T23:26:27.057+0200		10	10	n/a	n/a	0
2024-06-03T23:26:57.060+0200		0	0	10	10	0
```

The 10 events have been reprocessed successfully.

Here are the checkpoint info.

Before:

Current checkpoint info are:

| Partition | Sequence number | Offset |
| --------- | --------------- | ------ |
| 0         | 59              | 207680 |
| 1         | 49              | 172480 |
| 2         | 59              | 207680 |
| 3         | 39              | 137280 |

After:

Current checkpoint info are:

| Partition | Sequence number | Offset |
| --------- | --------------- | ------ |
| 0         | 59              | 207680 |
| 1         | 59              | 207680 |
| 2         | 59              | 207680 |
| 3         | 39              | 137280 |


Of the 10 events published, 

- 0  landed on partition 0
- 10 landed on partition 1 (49 > 59)
- 0  landed on partition 2
- 0  landed on partition 3
