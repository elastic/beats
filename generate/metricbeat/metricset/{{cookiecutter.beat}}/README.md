# {{cookiecutter.project_name}}

{{cookiecutter.project_name}} is a beat based on metricbeat which was generated with metricbeat/metricset generator.


## Getting started

To get started run:

```
make setup
make create-metricset
```

When running `make create-metricset` it will ask you for the module and metricset name. Insert the name accordingly.

To compile your beat run `make`. Then you can run the following command to see the first output:

```
{{cookiecutter.beat}} -e -d "*"
```
