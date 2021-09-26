# Dev Guide

### Debug the scripts processor

- Configure the filebeat to accept stdin input and output results to stdout
- Add your script processor

Like this:

```yaml
filebeat.inputs:
  - type: stdin
    multiline.type: pattern
    multiline.pattern: '^# Time: '
    multiline.negate: true
    multiline.match: after
    multiline.timeout: 1s

processors:
  - script:
      lang: javascript
      id: tidb_slow_log_parser
      params: { }
      source: >
        # your js scripts here
output.console:
  pretty: true

path.home: ./__local_home
logging.level: info
logging.metrics.enabled: false
```

Use this configuration to start filebeat process.

### Prepare a minimal elasticsearch cluster

Use docker-compose to start an elasticsearch instance and a kibana instance.

`docker-compose.yml`:

```yaml
version: '2.2'
services:
  es01:
    image: docker.elastic.co/elasticsearch/elasticsearch:7.14.1
    container_name: es01
    environment:
      - node.name=es01
      - cluster.name=es-docker-cluster
      - cluster.initial_master_nodes=es01
      - bootstrap.memory_lock=true
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    ulimits:
      memlock:
        soft: -1
        hard: -1
    volumes:
      - data01:/usr/share/elasticsearch/data
    ports:
      - 9200:9200
    networks:
      - elastic

  kib01:
    image: docker.elastic.co/kibana/kibana:7.14.1
    container_name: kib01
    ports:
      - 5601:5601
    environment:
      ELASTICSEARCH_URL: http://es01:9200
      ELASTICSEARCH_HOSTS: '["http://es01:9200"]'
    networks:
      - elastic

volumes:
  data01:
    driver: local

networks:
  elastic:
    driver: bridge
```

### Run Tests

```shell
# Just run once
make clean
make python-env
source ./build/python-env/bin/activate
make filebeat.test
# Run after each time module changing
make update
# Start tests
GENERATE=1 INTEGRATION_TESTS=1 BEAT_STRICT_PERMS=false TESTING_FILEBEAT_MODULES=tidb pytest tests/system/test_modules.py
```

### Get records from the elasticsearch instance

```shell
curl -X GET --location "http://localhost:9200/test-filebeat-modules/_search"
```

### View test configs and logs

Test results locate at `filebeat/build/system-tests/run/test_modules.Test.test_fileset_file_0_tidb/`
