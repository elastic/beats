Available scripts
-----------------


The following scripts are used by the unified release process:

| File                 | Description |
|----------------------|-------------|
| get_version          | Returns the current version |
| set_version          | Sets the current version in all places where change is required. Doesn't commit changes. |
| deploy               | Builds all artifacts for the officially supported Beats |



Other scripts:


| File                 | Description |
|----------------------|-------------|
| aggregate_coverage.py | Used to create coverage reports that contain both unit and system tests data |
| merge_pr | Used to make it easier to open a PR that merges one branch into another. |


Import / export the dashboards of a single Beat:

| File                  | Description |
|-----------------------|-------------|
| export_dashboards.py  | Python script to export the Beat dashboards from Elasticsearch to a local directory|

Running export_dashboards.py in environment
----------------------------------------------

If you are running the python script for the first time, you need to create the
environment by running the following commands in the `beats/dev-tools`
directory:

```
virtualenv env
. env/bin/activate
pip install -r requirements.txt
```

This creates the environment that contains all the python packages required to
run the `export_dashboards.py` script. Thus, for the next runs you just need
to enable the environment:

```
. env/bin/activate
```
