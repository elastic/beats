# MISP module

## Caveats

* Module is to be considered _beta_.

## How to try the module from source

You should already have MISP installed and running. Information about the MISP platform can be found here: https://www.circl.lu/doc/misp.

Build Filebeat

```
cd x-pack/filebeat
mage build update
./filebeat setup --modules=misp -e -d "*" -E 'setup.dashboards.directory=build/kibana'
```

Enable the MISP module

```
./filebeat modules enable misp
```

Start Filebeat

```
./filebeat -e
```

You can see the MISP Overview dashboard and the imported threat indicators in Kibana.
