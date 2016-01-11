# Testing environments

These environments are intended for manual testing. To start an environment run:


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

Often de default ip is `192.168.99.100`.


## Cleanup
In case your environment is messed up because of multiple instances still running and conflicting with each other, use the following commands to clean up. Please be aware that this will stop ALL docker containers ony our docker-machine.

```
make clean
```


## Notes

Every container has a name corresponding with the service. This requires to shut down an environment and clean it up before starting an other environment. This is intentional to prevent conflicts.
