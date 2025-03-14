---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-couchdb.html
---

# CouchDB fields [exported-fields-couchdb]

couchdb module


## couchdb [_couchdb]

Couchdb metrics


## server [_server_2]

Contains CouchDB server stats


## httpd [_httpd]

HTTP statistics

**`couchdb.server.httpd.view_reads`**
:   Number of view reads

type: long


**`couchdb.server.httpd.bulk_requests`**
:   Number of bulk requests

type: long


**`couchdb.server.httpd.clients_requesting_changes`**
:   Number of clients for continuous _changes

type: long


**`couchdb.server.httpd.temporary_view_reads`**
:   Number of temporary view reads

type: long


**`couchdb.server.httpd.requests`**
:   Number of HTTP requests

type: long



## httpd_request_methods [_httpd_request_methods]

HTTP request methods

**`couchdb.server.httpd_request_methods.COPY`**
:   Number of HTTP COPY requests

type: long


**`couchdb.server.httpd_request_methods.HEAD`**
:   Number of HTTP HEAD requests

type: long


**`couchdb.server.httpd_request_methods.POST`**
:   Number of HTTP POST requests

type: long


**`couchdb.server.httpd_request_methods.DELETE`**
:   Number of HTTP DELETE requests

type: long


**`couchdb.server.httpd_request_methods.GET`**
:   Number of HTTP GET requests

type: long


**`couchdb.server.httpd_request_methods.PUT`**
:   Number of HTTP PUT requests

type: long



## httpd_status_codes [_httpd_status_codes]

HTTP status codes statistics

**`couchdb.server.httpd_status_codes.200`**
:   Number of HTTP 200 OK responses

type: long


**`couchdb.server.httpd_status_codes.201`**
:   Number of HTTP 201 Created responses

type: long


**`couchdb.server.httpd_status_codes.202`**
:   Number of HTTP 202 Accepted responses

type: long


**`couchdb.server.httpd_status_codes.301`**
:   Number of HTTP 301 Moved Permanently responses

type: long


**`couchdb.server.httpd_status_codes.304`**
:   Number of HTTP 304 Not Modified responses

type: long


**`couchdb.server.httpd_status_codes.400`**
:   Number of HTTP 400 Bad Request responses

type: long


**`couchdb.server.httpd_status_codes.401`**
:   Number of HTTP 401 Unauthorized responses

type: long


**`couchdb.server.httpd_status_codes.403`**
:   Number of HTTP 403 Forbidden responses

type: long


**`couchdb.server.httpd_status_codes.404`**
:   Number of HTTP 404 Not Found responses

type: long


**`couchdb.server.httpd_status_codes.405`**
:   Number of HTTP 405 Method Not Allowed responses

type: long


**`couchdb.server.httpd_status_codes.409`**
:   Number of HTTP 409 Conflict responses

type: long


**`couchdb.server.httpd_status_codes.412`**
:   Number of HTTP 412 Precondition Failed responses

type: long


**`couchdb.server.httpd_status_codes.500`**
:   Number of HTTP 500 Internal Server Error responses

type: long



## couchdb [_couchdb_2]

couchdb statistics

**`couchdb.server.couchdb.database_writes`**
:   Number of times a database was changed

type: long


**`couchdb.server.couchdb.open_databases`**
:   Number of open databases

type: long


**`couchdb.server.couchdb.auth_cache_misses`**
:   Number of authentication cache misses

type: long


**`couchdb.server.couchdb.request_time`**
:   Length of a request inside CouchDB without MochiWeb

type: long


**`couchdb.server.couchdb.database_reads`**
:   Number of times a document was read from a database

type: long


**`couchdb.server.couchdb.auth_cache_hits`**
:   Number of authentication cache hits

type: long


**`couchdb.server.couchdb.open_os_files`**
:   Number of file descriptors CouchDB has open

type: long


