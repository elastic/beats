# Zeek (Bro) module

## Caveats

* Module is to be considered _alpha_.
* Field names will be changing for 7.0 to comply with Elastic Common Schema (ECS).

## How to try the module from source

Install Zeek/Bro (for MacOS with Brew)

```
brew install bro
```

Configure it to process network traffic and generate logs. 
Edit `/usr/local/etc/node.cfg` to use the proper network interfaces. 
Edit `/usr/local/etc/network.cfg` to specify local networks accordingly.
Set `redef LogAscii::use_json=T;` in `/usr/local/share/bro/site/local.bro` to use JSON output. 

Start Zeek/Bro
```
sudo broctl deploy
```

Install Zeek/Bro (for Ubuntu Linux)

```
apt install bro
apt install broctl
```

Configure it to process network traffic and generate logs. 
Edit `/etc/bro/node.cfg` to use the proper network interfaces. 
Edit `/etc/bro/network.cfg` to specify local networks accordingly.
Set `redef LogAscii::use_json=T;` in `/usr/share/bro/site/local.bro` to use JSON output. 

Start Zeek/Bro

```
sudo broctl deploy
```


Build Filebeat

```
git clone git@github.com:elastic/beats.git
cd beats/x-pack/filebeat
make mage
mage clean update
mage build
mage dashboards
```

Update filebeat.yml to point to Elasticsearch and Kibana. Setup Filebeat.

```
./filebeat setup -e
```

Enable the Filebeat zeek module

```
./filebeat setup --modules zeek -e
./filebeat setup --modules zeek --dashboards -E setup.dashboards.directory=build/kibana
./filebeat modules enable zeek
```

Start Filebeat

```
./filebeat -e
```

Now, you should see the Zeek logs and dashboards in Kibana.
