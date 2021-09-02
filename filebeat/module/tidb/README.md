# Dev Guide

### Debugging scripts processors

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

### Preparations

```shell
docker run -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" docker.elastic.co/elasticsearch/elasticsearch:7.14.0
curl -X PUT --location "http://localhost:9200/_cluster/settings" \
    -H "Content-Type: application/json" \
    -d "{
          \"transient\": {
            \"logger.org.elasticsearch.cluster\": \"DEBUG\"
          }
        }"
make clean
make python-env
source ./build/python-env/bin/activate
```

### Running Tests

```shell
make update
make filebeat.test
GENERATE=1 INTEGRATION_TESTS=1 BEAT_STRICT_PERMS=false TESTING_FILEBEAT_MODULES=tidb TESTING_FILEBEAT_FILESETS=tidb pytest tests/system/test_modules.py
```

### Getting records in elasticsearch

```shell
curl -X GET --location "http://localhost:9200/test-filebeat-modules/_search"
```

### View test configs and runtime logs

`filebeat/build/system-tests/run/test_modules.Test.test_fileset_file_0_tidb/`
