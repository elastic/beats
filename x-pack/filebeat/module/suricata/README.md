# Suricata module

## Caveats

* Original Suricata event shoved as is `suricata.eve.`

## How to try the module from source

Build Filebeat

```
cd x-pack/filebeat
make mage
mage build update
./filebeat setup --modules=suricata -e -d "*" -c filebeat.yml -E 'setup.dashboards.directory=build/kibana'
```

Install Suricata (for MacOS with Brew)

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
./filebeat --modules=suricata -e -d "*" -c filebeat.yml
```

You can look for the Suricata saved searches and dashboards in Kibana.
