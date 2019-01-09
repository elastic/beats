# Bro/Zeek module

## Caveats

* Module is to be considered _alpha_.
* Field names will be changing for 7.0 to comply with Elastic Common Schema (ECS).

## How to try the module from source

Build Filebeat

```
cd x-pack/filebeat
make mage
mage build update
./filebeat setup --modules=bro -e -d "*" -c filebeat.yml -E 'setup.dashboards.directory=_meta/kibana'
```

Install Bro (for MacOS with Brew)

```
brew install bro
```

Configure Bro to process network traffic and generate logs. Edit `/usr/local/etc/node.cfg` to use the proper network interfaces. And set `redef LogAscii::use_json=T;` in `/usr/local/share/bro/site/local.bro` to use JSON output. 

Deploy Bro

```
sudo broctl deploy
```

Enable the Filebeat Bro module

```
./filebeat modules enable bro
```

You can look for the Bro saved searches and dashboards in Kibana.
