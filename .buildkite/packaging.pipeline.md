### Beats Packaging pipeline
[Buildkite packaging pipeline](https://buildkite.com/elastic/beats-packaging-pipeline) is used to build and publish the packages for the Beats. The pipeline is triggered by a commit to the `main` or release branches.
The pipeline definition is located in the `.buildkite/packaging.pipeline.yml`

### Triggers
Staging packaging DRA is triggered for release branches only.
Snapshot is triggered for `main` and release branches.

### Pipeline steps

#### Beats dashboard
Generates `build/distributions/dependencies.csv` and `tar.gz` and adds them to the `beats-dashboards` artifact. The CSV is required by the DRA prep step, which renames it to `dependencies-<version>.csv` for the `elastic/dra-prep` plugin's CSV classifier.

#### Packaging snapshot/staging

- Builds the Beats packages for all supported platforms and architectures (`mage package, mage ironbank`)
- Copies artifacts `build/distributions/<beat>/` directory and adds it as an artifact, where `<beat>` is the corresponding beat name.
- x-pack artifacts are also copied to `build/distributions/<beat>/` directory, where `<beat>` is the name of the beat. For example, `auditbeat`, not `x-pack/auditbeat`. The `x-pack/` prefix is stripped by `.buildkite/scripts/packaging/package-dra.sh`, so the DRA publish step sees one flat set of per-beat directories.

#### DRA publish
For `main` and release branches, this group resolves the stack version with `.buildkite/scripts/packaging/resolve-dra-version.sh` and then dynamically uploads `.buildkite/dra-prep-pipeline.yml` into the same build, once per workflow (snapshot and staging can both run in one build).

That sub-pipeline:

- Runs `.buildkite/scripts/stage-dra-artifacts.sh`, which downloads this build's `build/**/*` artifacts, renames `dependencies.csv` to `dependencies-<version>.csv`, rewrites `*.docker.tar.gz` names to the unified-release `docker-image-<platform>.tar.gz` form, and moves the workflow's own files into `artifacts/` (snapshot filenames carry `-SNAPSHOT`, staging ones do not).
- Publishes them with the `elastic/dra-prep` plugin.
- Annotates the build with a link to the published summary via `.buildkite/scripts/dra-annotate.sh`, and triggers `unified-release-dra-processing`.

Setting `DRY_RUN=true` in the Buildkite UI runs the prep step without uploading, and skips the annotation and the processing trigger.


