# Airflow metricbeat module

## How to test
You don't need a running Airflow instance in order to test the module:
sending a `statsd` metric (https://github.com/statsd/statsd/blob/master/docs/metric_types.md) to the `statsd` listener started by the module is enough:

```
$ ./metricbeat modules enable airflow
$ ./metricbeat -e # (Start metricbeat according to your preferred setup)
$ echo "dagrun.duration.failed.dagid:200|ms" > /dev/udp/127.0.0.1/8126 # (with any of the metrics that can be found at https://airflow.apache.org/docs/apache-airflow/stable/logging-monitoring/metrics.html)
```

### Testing with Airflow
You need to install `statsd` module in Airflow and let it point to host and port defined in the module:
https://airflow.apache.org/docs/apache-airflow/2.0.0/logging-monitoring/metrics.html

```
$ pip install 'apache-airflow[statsd]'
```

Add the following lines to your configuration file e.g. `airflow.cfg` ensuring `statsd_prefix` is left empty:

```
[metrics]
statsd_on = True
statsd_host = localhost
statsd_port = 8126
statsd_prefix =
```
