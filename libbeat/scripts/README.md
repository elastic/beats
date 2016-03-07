The script directory contains various scripts which exist to automate everything around beats.
Below is a brief description of each file / folder.


| File / Folder        | Description                                                                                                                                            |
|----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------|
| docker-entrypoint.sh | Entrypoint file used for the Dockerfile                                                                                                                |
| Makefile             | General Makefile which is copied over to all beats. This contains the basic methods which are shared across all beat                                   |
| update.sh            | This scripts brings a beat up-to-date based on the files in libbeat. This script should only be executed in a beat itself through running make update. |
| install-go.ps1       | PowerShell script for automating the install of Go on Windows.|


## Kibana Files

There exist two scripts, one to load dashboards into kibana to update existing dashboards and one to export dashboards and store the in the repository.

To create dashboards with visualisations and search it is important that always a fresh elasticsearch / kibana setup with the most recent version is used. To start a fresh environment with the latest version run `make start` inside `testing/environments`. This will start an environment with ES and Kibana and is accessible und your docker-machine ip (often 192.168.99.100:5601). Also point the script to these url to load or export the kibana files:

Export the kibana files:
```
python kibana_export.py --url http://192.168.99.100:5601/ --dir filebeat/etc/kibana
```

Import the kibana files into Kibana:
```
python kibana_import.py --url http://192.168.99.100:5601/ --dir filebeat/etc/kibana
```

When working with kibana files, make sure to only work on the files of one beat at the time and clean the environment when starting to work with an other beat.
