# Shipper

The Shipper output sends events to a shipper service via gRPC.

WARNING: This output is experimental and is not supposed to be used by anyone other than developers.

To use this output, edit the beat's configuration file and enable the shipper output by adding `output.shipper`.

Example configuration:

```yaml
output.shipper:
  server: "localhost:50051"
  ssl:
    enabled: true
    certificate_authorities: ["/etc/client/ca.pem"]
    certificate: "/etc/client/cert.pem"
    ssl.key: "/etc/client/cert.key"
  timeout: 30
  max_retries: 3
  bulk_max_size: 50
  ack_polling_interval: '5ms'
  backoff:
    init: 1
    max: 60
```

## Configuration options

You can specify the following `output.shipper` options in the beat's config file:

### `server`

The address of the gRPC server where the shipper service is hosted.

### `ssl`

Configuration options for SSL parameters like the root CA for gRPC connections.
See [configuration-ssl](https://www.elastic.co/guide/en/beats/filebeat/current/configuration-ssl.html) for more information.

### `timeout`

The number of seconds to wait for responses from the gRPC server before timing
out. The default is 30 (seconds).

### `max_retries`

The number of times to retry publishing an event after a publishing failure.
After the specified number of retries, the events are typically dropped.

Set `max_retries` to a value less than 0 to retry until all events are published.

The default is 3.

### `bulk_max_size`

The maximum number of events to buffer internally during publishing. The default is 50.

Specifying a larger batch size may add some latency and buffering during publishing.

Setting `bulk_max_size` to values less than or equal to 0 disables the
splitting of batches. When splitting is disabled, the queue decides on the
number of events to be contained in a batch.

### `ack_polling_interval`

The minimal interval for getting persisted index updates from the shipper server. Batches of events are acknowledged asynchronously in the background. If after the `ack_polling_interval` duration the persisted index value changed all batches pending acknowledgment will be checked against the new value and acknowledged if `persisted_index` >= `accepted_index`.

The default value is `5ms`, cannot be set to a value less then the default.

### `backoff.init`

The number of seconds to wait before trying to republish to the shipper
after a network error. After waiting `backoff.init` seconds, {beatname_uc}
tries to republish. If the attempt fails, the backoff timer is increased
exponentially up to `backoff.max`. After a successful publish, the backoff
timer is reset. The default is 1s.

### `backoff.max`

The maximum number of seconds to wait before attempting to republish to
the shipper after a network error. The default is 60s.
