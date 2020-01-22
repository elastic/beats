# Testing environments

These environments are intended for manual and automated testing. The docker-compose files can be combined to create the different environment.


# Manual testing

The different environments can be started with the following commands for manual testing. These environments expose ports of Elasticsearch, Logstash and Kibana on the Docker-Machine ip.

Running the environment chains the following docker-compose files together

* local.yml: Definition of ports which have to be exposed for local testing including kibana
* latest.yml: Latest version of elasticsearch, logstash, kibana
* snapshot.yml: Snapshot version of elasticsearch, logstash, kibana


## Start / Stop environment

```
make start ENV=es17-ls15-kb41.yml
```

This will start the environment and log you into the debian machine. This machine is intended for manual testing of the beats. Download the beats package or snapshot you want to test. Elasticsearch can be reached under the host `elasticsearch`, logstash under `logstash`. Make sure to update the configuration file of the beat with the specific host.

To stop an clean up the environment afterwards, make sure to run:

```
make stop ENV=es17-ls15-kb41.yml
```


## Update containers

As for testing, some default installation must be changed, access to the containers is needed. Each container has a unique name which corresponds with the service name. To access a running container of elasticsearch, run:

```
docker exec -it elasticsearch bash
```

## Access machines from external

It is useful to sometimes access the containers from a browser, especially for Kibana. Elasticsearch exposes port 9200 and Kibana 5601. Make sure no other services on your machine are already assigned to these ports. To access Kibana for example, go to the following url:

```
http://docker-machine-ip:5601/
```

Often the default address is `localhost`.


## Cleanup
In case your environment is messed up because of multiple instances still running and conflicting with each other, use the following commands to clean up. Please be aware that this will stop ALL docker containers ony our docker-machine.

```
make clean
```


## Notes

Every container has a name corresponding with the service. This requires to shut down an environment and clean it up before starting an other environment. This is intentional to prevent conflicts.


# Automated Testing

These environments are also used for integration testing in the different beats. For this, `make testsuite` by default uses the snapshot environment. To select a different environment during testing, run the following command to use the latest environment:

```
TESTING_ENVIRONMENT=latest make testsuite
```

This will run the full testsuite but with latest environments instead of snapshot.


## Defaults

By default, elasticsearch, logstash and kibana are started. These are available at all time that these environments are used. Running the environment, chains the following docker-compose flies together:

* snapshot.yml: Snapshot version of elasticsearch, logstash, kibana
* docker-compose.yml: Local beat docker-compose file


## Updating environments

If the snapshot environment is updated with a new build, all beats will automatically build with the most recent version.
