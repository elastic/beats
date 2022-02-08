# Elastic Agent developer docs

The source files for the general Elastic Agent documentation are currently stored
in the [observability-docs](https://github.com/elastic/observability-docs) repo. The following docs are only focused on getting developers started building code for Elastic Agent.

## Testing docker container

Running Elastic Agent in a docker container is a common use case. To build the Elastic Agent and create a docker image run the following command:

```
DEV=true SNAPSHOT=true PLATFORMS=linux/amd64 TYPES=docker mage package
```

If you are in the 7.13 branch, this will create the `docker.elastic.co/beats/elastic-agent:7.13.0-SNAPSHOT` image in your local environment. Now you can use this to for example test this container with the stack in elastic-package:

```
elastic-package stack up --version=7.13.0-SNAPSHOT -v
```

Please note that the docker container is built in both standard and 'complete' variants.
The 'complete' variant contains extra files, like the chromium browser, that are too large
for the standard variant.
