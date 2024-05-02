### Beats Packaging pipeline
[Buildkite packaging pipeline](https://buildkite.com/elastic/beats-packaging-pipeline) is used to build and publish the packages for the Beats. The pipeline is triggered by a commit to the `main` or release branches.
The pipeline definition is located in the `.buildkite/packaging.pipeline.yml`

### Triggers
Staging packaging DRA is triggered for release branches only.
Snapshot is triggered for `main` and release branches.

### Pipeline steps

#### Beats dashboard
Generates `build/distributions/dependencies.csv` and `tar.gz` and adds them to the `beats-dashboards` artifact. This is required by the release-manager configuration.

#### Packaging snapshot/staging

- Builds the Beats packages for all supported platforms and architectures (`mage package, mage ironbank`)
- Copies artifacts `build/distributions/<beat>/` directory and adds it as an artifact, where `<beat>` is the corresponding beat name.
- x-pack artifacts are also copied to `build/distributions/<beat>/` directory, where `<beat>` is the name of the beat. For example, `auditbeat`, not `x-pack/auditbeat`. It's required for the DRA publish step by [release-manager configuration](https://github.com/elastic/infra/blob/master/cd/release/release-manager/project-configs/master/beats.gradle).

#### DRA publish
Downloads the artifacts from the `packaging snapshot/staging` step and publishes them to the Elastic DRA registry.


