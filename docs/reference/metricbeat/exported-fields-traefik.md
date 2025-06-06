---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-traefik.html
---

<!-- This file is generated! See scripts/generate_fields_docs.py -->

# Traefik fields [exported-fields-traefik]

Traefik reverse proxy / load balancer metrics

## traefik [_traefik]

Traefik reverse proxy / load balancer metrics

## health [_health]

Metrics obtained from Traefik's health API endpoint

**`traefik.health.uptime.sec`**
:   Uptime of Traefik instance in seconds

type: long


## response [_response]

Response metrics

**`traefik.health.response.count`**
:   Number of responses

type: long


**`traefik.health.response.avg_time.us`**
:   Average response time in microseconds

type: long


**`traefik.health.response.status_codes.*`**
:   Number of responses per status code

type: object


