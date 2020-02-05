For manual testing and development of this module, docker can be used.

The following docker compose starts a two node cluster, dbprod-02 service
can be copied and renamed many times to have more servers:
```
services:
  dbprod-01:
    image: cockroachdb/cockroach:v19.1.1
    command: start --insecure --advertise-addr dbprod-01

  dbprod-02:
    image: cockroachdb/cockroach:v19.1.1
    command: start --insecure --advertise-addr dbprod-02 --join dbprod-01
```

And this configuration can be used for metricbeat:
```
metricbeat.autodiscover.providers:
  - type: docker
    templates:
      - condition:
          contains:
            docker.container.image: cockroachdb
        config:
          - module: cockroachdb
            hosts: ['${data.host}:8080']
```

To generate data and load, these examples are pretty handy: https://github.com/cockroachdb/examples-go
