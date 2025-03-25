---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-nginx.html
---

# Nginx fields [exported-fields-nginx]

Nginx server status metrics collected from various modules.


## nginx [_nginx]

`nginx` contains the metrics that were scraped from nginx.


## stubstatus [_stubstatus]

`stubstatus` contains the metrics that were scraped from the ngx_http_stub_status_module status page.

**`nginx.stubstatus.hostname`**
:   Nginx hostname.

type: keyword


**`nginx.stubstatus.active`**
:   The current number of active client connections including Waiting connections.

type: long


**`nginx.stubstatus.accepts`**
:   The total number of accepted client connections.

type: long


**`nginx.stubstatus.handled`**
:   The total number of handled client connections.

type: long


**`nginx.stubstatus.dropped`**
:   The total number of dropped client connections.

type: long


**`nginx.stubstatus.requests`**
:   The total number of client requests.

type: long


**`nginx.stubstatus.current`**
:   The current number of client requests.

type: long


**`nginx.stubstatus.reading`**
:   The current number of connections where Nginx is reading the request header.

type: long


**`nginx.stubstatus.writing`**
:   The current number of connections where Nginx is writing the response back to the client.

type: long


**`nginx.stubstatus.waiting`**
:   The current number of idle client connections waiting for a request.

type: long


