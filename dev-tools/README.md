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
| import_dashboards.sh  | Bash script to import the Beat dashboards from a local directory in Elasticsearch |
| import_dashboards.ps1 | Powershell script to import the Beat dashboards from a local directory in Elasticsearch |
| export_dashboards.py  | Python script to export the Beat dashboards from Elasticsearch to a local directory|


