---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-uwsgi.html
---

# uWSGI fields [exported-fields-uwsgi]

uwsgi module


## uwsgi [_uwsgi]


## status [_status_8]

uwsgi.status metricset fields

**`uwsgi.status.total.requests`**
:   Total requests handled

type: long


**`uwsgi.status.total.exceptions`**
:   Total exceptions

type: long


**`uwsgi.status.total.write_errors`**
:   Total requests write errors

type: long


**`uwsgi.status.total.read_errors`**
:   Total read errors

type: long


**`uwsgi.status.total.pid`**
:   Process id

type: long


**`uwsgi.status.worker.id`**
:   Worker id

type: long


**`uwsgi.status.worker.pid`**
:   Worker process id

type: long


**`uwsgi.status.worker.accepting`**
:   State of worker, 1 if still accepting new requests otherwise 0

type: long


**`uwsgi.status.worker.requests`**
:   Number of requests served by this worker

type: long


**`uwsgi.status.worker.delta_requests`**
:   Number of requests served by this worker after worker is reloaded when reached MAX_REQUESTS

type: long


**`uwsgi.status.worker.exceptions`**
:   Exceptions raised

type: long


**`uwsgi.status.worker.harakiri_count`**
:   Dropped requests by timeout

type: long


**`uwsgi.status.worker.signals`**
:   Emitted signals count

type: long


**`uwsgi.status.worker.signal_queue`**
:   Number of signals waiting to be handled

type: long


**`uwsgi.status.worker.status`**
:   Worker status (cheap, pause, sig, busy, idle)

type: keyword


**`uwsgi.status.worker.rss`**
:   Resident Set Size. memory currently used by a process. if always zero try `--memory-report` option of uwsgi

type: long


**`uwsgi.status.worker.vsz`**
:   Virtual Set Size. memory size assigned to a process. if always zero try `--memory-report` option of uwsgi

type: long


**`uwsgi.status.worker.running_time`**
:   Process running time

type: long


**`uwsgi.status.worker.respawn_count`**
:   Respawn count

type: long


**`uwsgi.status.worker.tx`**
:   Transmitted size

type: long


**`uwsgi.status.worker.avg_rt`**
:   Average response time

type: long


**`uwsgi.status.core.id`**
:   worker ID

type: long


**`uwsgi.status.core.worker_pid`**
:   Parent worker PID

type: long


**`uwsgi.status.core.requests.total`**
:   Number of total requests served

type: long


**`uwsgi.status.core.requests.static`**
:   Number of static file serves

type: long


**`uwsgi.status.core.requests.routed`**
:   Routed requests

type: long


**`uwsgi.status.core.requests.offloaded`**
:   Offloaded requests

type: long


**`uwsgi.status.core.write_errors`**
:   Number of failed writes

type: long


**`uwsgi.status.core.read_errors`**
:   Number of failed reads

type: long


