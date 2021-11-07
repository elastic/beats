# NOAA [![slack.cloudfoundry.org][slack-badge]][loggregator-slack]

[![Concourse Status](https://loggregator.ci.cf-app.com/api/v1/pipelines/submodules/jobs/noaa-unit-tests/badge)](https://loggregator.ci.cf-app.com/teams/main/pipelines/submodules/jobs/noaa-unit-tests)
[![Coverage Status](https://coveralls.io/repos/cloudfoundry/noaa/badge.png)](https://coveralls.io/r/cloudfoundry/noaa)
[![GoDoc](https://godoc.org/github.com/cloudfoundry/noaa?status.png)](https://godoc.org/github.com/cloudfoundry/noaa)

noaa is a client library to consume metric and log messages from Doppler.

## Get the Code

This Go project is designed to be imported into `$GOPATH`, rather than being cloned into any working directory. There are two ways to do this.

- The easiest way with with `go get`. This will import the project, along with all dependencies, into your `$ GOPATH`.
  ```
  $ echo $GOPATH
  /Users/myuser/go

  $ go get github.com/cloudfoundry/noaa

  $ ls ~/go/src/github.com/cloudfoundry/
  noaa/         sonde-go/
  ```

- You can also manually clone the repo into your `$GOPATH`, but you then have to manually import dependencies.
  ```
  $ echo $GOPATH
  /Users/myuser/go

  $ cd /Users/myuser/go/src/github.com/cloudfoundry
  $ git clone git@github.com:cloudfoundry/noaa.git
  $ cd noaa
  $ go get ./...
  ```

## Updates

### Reconnecting to Traffic Controller

noaa has recently updated its reconnect strategy from trying to reconnect five
times in quick succession to a back-off strategy. The back-off strategy can be
configured by setting the [SetMinRetryDelay()](https://godoc.org/github.com/cloudfoundry/noaa/consumer#Consumer.SetMinRetryDelay)
and the [SetMaxRetryDelay()](https://godoc.org/github.com/cloudfoundry/noaa/consumer#Consumer.SetMaxRetryDelay).

During reconnection, noaa will wait initially at the `MinRetryDelay` interval
and double until it reaches `MaxRetryDelay` where it will try reconnecting
indefinitely at the `MaxRetryDelay` interval.

This behavior will affect functions like `consumer.Firehose()`, `consumer.Stream()`
and `consumer.TailingLogs()`.

## Sample Applications

### Prerequisites

In order to use the sample applications below, you will have to export the
following environment variables:

* `CF_ACCESS_TOKEN` - You can get this value by executing (`$ cf oauth-token`).
  Example:

```bash
export CF_ACCESS_TOKEN="bearer eyJhbGciOiJSUzI1NiJ9.eyJqdGkiOiI3YmM2MzllOC0wZGM0LTQ4YzItYTAzYS0xYjkyYzRhMWFlZTIiLCJzdWIiOiI5YTc5MTVkOS04MDc1LTQ3OTUtOTBmOS02MGM0MTU0YTJlMDkiLCJzY29wZSI6WyJzY2ltLnJlYWQiLCJjbG91ZF9jb250cm9sbGVyLmFkbWluIiwicGFzc3dvcmQud3JpdGUiLCJzY2ltLndyaXRlIiwib3BlbmlkIiwiY2xvdWRfY29udHJvbGxlci53cml0ZSIsImNsb3VkX2NvbnRyb2xsZXIucmVhZCJdLCJjbGllbnRfaWQiOiJjZiIsImNpZCI6ImNmIiwiZ3JhbnRfdHlwZSI6InBhc3N3b3JkIiwidXNlcl9pZCI6IjlhNzkxNWQ5LTgwNzUtNDc5NS05MGY5LTYwYzQxNTRhMmUwOSIsInVzZXJfbmFtZSI6ImFkbWluIiwiZW1haWwiOiJhZG1pbiIsImlhdCI6MTQwNDg0NzU3NywiZXhwIjoxNDA0ODQ4MTc3LCJpc3MiOiJodHRwczovL3VhYS4xMC4yNDQuMC4zNC54aXAuaW8vb2F1dGgvdG9rZW4iLCJhdWQiOlsic2NpbSIsIm9wZW5pZCIsImNsb3VkX2NvbnRyb2xsZXIiLCJwYXNzd29yZCJdfQ.mAaOJthCotW763lf9fysygqdES_Mz1KFQ3HneKbwY4VJx-ARuxxiLh8l_8Srx7NJBwGlyEtfYOCBcIdvyeDCiQ0wT78Zw7ZJYFjnJ5-ZkDy5NbMqHbImDFkHRnPzKFjJHip39jyjAZpkFcrZ8_pUD8XxZraqJ4zEf6LFdAHKFBM"
```

* `DOPPLER_ADDR` - It is based on your environment. Example:

```bash
export DOPPLER_ADDR="wss://doppler.10.244.0.34.xip.io:4443"
```


### Application logs

The `samples/app_logs/main.go` application streams logs for a particular app.
The following environment variable needs to be set:

* `APP_GUID` - You can get this value from running `$ cf app APP --guid`.
  Example:

```
export APP_GUID=55fdb274-d6c9-4b8c-9b1f-9b7e7f3a346c
```

Then you can run the sample app like this:

```
go build -o bin/app_logs samples/app_logs/main.go
bin/app_logs
```

### Logs and metrics firehose

The `samples/firehose/main.go` application streams metrics data and logs for
all apps.

You can run the firehose sample app like this:

```
go build -o bin/firehose samples/firehose/main.go
bin/firehose
```

Multiple subscribers may connect to the firehose endpoint, each with a unique
subscription_id (configurable in `main.go`). Each subscriber (in practice, a
pool of clients with a common subscription_id) receives the entire stream. For
each subscription_id, all data will be distributed evenly among that
subscriber's client pool.

### Container metrics

The `samples/container_metrics/consumer/main.go` application streams container
metrics for the specified appId.

You can run the container metrics sample app like this:

```
go build -o bin/container_metrics samples/container_metrics/consumer/main.go
bin/container_metrics
```

For more information to setup a test environment in order to pull container
metrics look at the README.md in the container_metrics sample.

## Development

Use `go get -d -v -t ./... && ginkgo --race --randomizeAllSpecs --failOnPending --skipMeasurements --cover` to
run the tests.


[slack-badge]:          https://slack.cloudfoundry.org/badge.svg
[loggregator-slack]:    https://cloudfoundry.slack.com/archives/loggregator
