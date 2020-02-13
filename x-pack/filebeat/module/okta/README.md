# OKTA module

## Caveats

* Module is to be considered _beta_.

## How to try the module from distribution install


```
./filebeat setup --modules=okta -e --dashboards
```

Enable the OKTA module

```
./filebeat modules enable okta
```

Start Filebeat

```
./filebeat -e
```

You can see the OKTA Overview dashboard in Kibana.
