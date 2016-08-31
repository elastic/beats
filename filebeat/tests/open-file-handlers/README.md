# Filebeat long running tests

These tests are designed to run over a longer period and detect potential issues with filebeat like open file handler.

The following test run filebeat inside a docker container and read the log files created by other docker containers. Inside the docker container a metricbeat instance is started to monitor filebeat and the open file handlers.

The log files are created by python script. To change the number of events that is created, either the python script can be adapted or `docker-compose scale logs=4` can be used to start / stop logging containers.

# Setup

To start the "testing" use `make start`. To have filebeat and metricbeat send events to a remote host, pass host, username and password as following:

```
ES_HOST=http://localhost:9200 ES_USER=admin ES_PASSWORD=password docker-compose build
```

To stop the environment and clean it up, use `make stop`

# Timelion

To visualise the open file handlers in Timelion, the following query can be used:

```
.es(*,metric=max:beats.filebeat.harvesters.files.open) .es(*,metric=max:system.process.fd.open) .es(*,metric=max:system.process.fd.open).subtract(.es(*,metric=max:beats.filebeat.harvesters.files.open) )
```
