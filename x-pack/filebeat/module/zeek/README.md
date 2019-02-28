# Zeek (Bro) module

## Caveats

* Module is to be considered _beta_.

## Install and Configure Zeek/Bro

### Install Zeek/Bro (for MacOS with Brew)

```
brew install bro
```

* Configure it to process network traffic and generate logs. 
* Edit `/usr/local/etc/node.cfg` to use the proper network interfaces. 
* Edit `/usr/local/etc/networks.cfg` to specify local networks accordingly.
* Set `redef LogAscii::use_json=T;` in `/usr/local/share/bro/site/local.bro` to use JSON output. 

### Install Zeek/Bro (for Ubuntu Linux)

```
apt install bro
apt install broctl
```

* Configure it to process network traffic and generate logs. 
* Edit `/etc/bro/node.cfg` to use the proper network interfaces. 
* Edit `/etc/bro/networks.cfg` to specify local networks accordingly.
* Set `redef LogAscii::use_json=T;` in `/usr/share/bro/site/local.bro` to use JSON output. 

## Start Zeek/Bro

```
sudo broctl deploy
```

## Download and install Filebeat

Grab the filebeat binary from elastic.co, and install it by following the instructions.

## Configure Filebeat module and run

Update filebeat.yml to point to Elasticsearch and Kibana. Setup Filebeat.

```
./filebeat setup --modules zeek -e -E 'setup.dashboards.enabled=true'
```

Enable the Filebeat zeek module

```
./filebeat modules enable zeek
```

Start Filebeat

```
./filebeat -e
```

Now, you should see the Zeek logs and dashboards in Kibana.
