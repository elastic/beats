For manual testing and development of this module, start metricbeat with the standard configuration for the module:

```
------------------------------------------------------------------------------
- module: statsd
  metricsets: ["server"]
  host: "localhost"
  port: "8125"
  enabled: true
  #ttl: "30s"
------------------------------------------------------------------------------
```

Look for a log line to this effect:
```
Started listening for UDP on: 127.0.0.1:8125
```

then use a statsd client to test the features. In an empty directory do the following:

```
$ npm install statsd-client
$ node
> var SDC = require('statsd-client'),
    sdc = new SDC({host: 'localhost', port: 8125});

> sdc.increment('systemname.subsystem.value'); // Increment by one
> sdc.gauge('what.you.gauge', 100);
> sdc.gaugeDelta('what.you.gauge', -70); // Will now count 50
> sdc.gauge('gauge.with.tags', 100, {foo: 'bar'});
> sdc.set('set.with.tags', 100, {foo: 'bar'});
> sdc.set('set.with.tags', 200, {foo: 'bar'});
> sdc.set('set.with.tags', 100, {foo: 'baz'});
....

<CTRL+D>
```
