# Overview

These are the build and test files for the Observability Heartbeat to generate and validate the IronBank Docker images.

## Docker context generation

The docker context generation is done as part of the `packaging` pipeline.

## Docker context validation

It has been decoupled from the generation. It requires the below steps to generate the required artifacts and validate the docker context can be built.

```bash
cd x-pack/heartbeat
make -C ironbank package
mage ironbank
make -C ironbank validate-ironbank
```

If for any reason it failed to be built, it might be related to some
dependencies that have been changed and hence it's required to update them in `dev-tools/packaging/templates/ironbank/heartbeat/hardening_manifest.yaml` accordingly.

These steps are explained in an internal GitHub repository, and for the time being won't be public available.
