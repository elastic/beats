### Beats Packaging pipeline
[Buildkite packaging pipeline](https://buildkite.com/elastic/beats-packaging-pipeline) is used to build and publish the packages for the Beats. The pipeline is triggered by a commit to the `main` or release branches.
The pipeline definition is located in the `.buildkite/packaging.pipeline.yml`

### Triggers
Staging packaging DRA is triggered for the `main` and release branches.
Snapshot can be triggered for any branch by the `/package` comment in the PR. The release-manager dry-run will be used for PR builds.

### Pipeline steps

#### Beats dashboards

Generates `build/distributions/dependencies.csv` and adds it to the `beats-dashboards` artifact. The `dependencies.csv` is required by the release-manager configuration

#### Packaging snapshot/staging

- Builds the Beats packages for all supported platforms and architectures (`mage package, mage ironbank`)
- Copies artifacts `build/distributions/<beat>/` directory and adds it as an artifact. Where `<beat>` is the name of the beat
- x-pack artifacts a also copied to `build/distributions/<beat>/` directory, where `<beat>` is the name of the beat. For example, `auditbeat`, not `x-pack/auditbeat`

#### DRA publish
Downloads the artifacts from the `packaging snapshot/staging` step and publishes them to the Elastic DRA registry.


