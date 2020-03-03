### Development

Run metricbeat locally with configured `statsd` module:
 
```yaml
- module: statsd
  metricsets: ["server"]
  host: "localhost"
  port: "8125"
  enabled: true
  #ttl: "30s"
```
 
Use favorite statsd client to emit metrics, e.g.:

```bash
$ npm install statsd-client
$ node
```

Emit some metrics:

```javascript
let SDC = require('statsd-client'), sdc = new SDC({host: 'localhost', port: 8125});
sdc.increment('systemname.subsystem.value');
sdc.gauge('what.you.gauge', 100);
sdc.gaugeDelta('what.you.gauge', -70);
sdc.gauge('gauge.with.tags', 100, {foo: 'bar'});
sdc.set('set.with.tags', 100, {foo: 'bar'});
sdc.set('set.with.tags', 200, {foo: 'bar'});
sdc.set('set.with.tags', 100, {foo: 'baz'});
```
