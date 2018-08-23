# Suricata module

Shove the content of this repo under `filebeats/module/suricata` for testing,
until Filebeats supports loading modules from `x-pack/filebeats/module`
(issue [beats#7524](https://github.com/elastic/beats/issues/7524)).

## Caveats with this preliminary version

* Dashboards, visualizations and saved searches are trivial at this time.
* Original Suricata event shoved as is `suricata.eve.`
* Due to limitations in Ingest Node field copying, all events have `ecs.user_agent`.
  Events that actually have user agent information are missing the `ecs.user_agent.raw`,
  and all of their ua fields have the value "Other". Disregard those.
* GeoIP is done twice. Once on ECS fields and once on original fields, saving back
  under original object (in order to have usable geo\_points).
* ECS
  * ECS fields nested under `ecs.` instead of being at the top level,
    to avoid clashes during development.
  * ECS fields are not set in the index template (and so are simply detected by ElasticSearch),
    singe beats modules can only configure fields under their own section
    (in this case `suricata.eve.*`)

## How to try the module

Copy this full repo at `beats/filebeat/module/suricata`.

Set up the module (you may have to delete your Filebeat index template first).

```
cd filebeat
make update
./filebeat setup --modules=suricata -e -d "*" -c your/filebeat.yml -E 'setup.dashboards.directory=_meta/kibana'
```

Install Suricata

```
brew install suricata --with-jansson
```

Configure it to generate the EVE JSON log. Edit `/usr/local/etc/suricata/suricata.yaml` and set

```
- eve-log:
    enabled: yes
```

Start Suricata

```
sudo suricata -i en0 # optionally more -i en1 -i en2...
```

Start the Suricata Filebeat module

```
./filebeat --modules=suricata -e -d "*" -c your/filebeat.yml
```

You can look for the Suricata saved searches and dashboards in Kibana.
