## 9.5.1 [beats-release-notes-9.5.1]





### Fixes [beats-9.5.1-fixes]


**All**

* Fix unknown type for map[string]string in otelmap. [#52271](https://github.com/elastic/beats/pull/52271) [#52410](https://github.com/elastic/beats/pull/52410) [#52459](https://github.com/elastic/beats/pull/52459) [#52420](https://github.com/elastic/beats/pull/52420) [#52488](https://github.com/elastic/beats/pull/52488) [#52515](https://github.com/elastic/beats/pull/52515) [#52460](https://github.com/elastic/beats/issues/52460)

**Filebeat**

* Fix offset commits when using multiline in Journald and Kafka inputs. [#52271](https://github.com/elastic/beats/pull/52271) [#52410](https://github.com/elastic/beats/pull/52410) [#52459](https://github.com/elastic/beats/pull/52459) [#52420](https://github.com/elastic/beats/pull/52420) [#52488](https://github.com/elastic/beats/pull/52488) [#52515](https://github.com/elastic/beats/pull/52515) [#51981](https://github.com/elastic/beats/issues/51981)

**Heartbeat**

* Bake Synthetics browser binaries into the heartbeat Docker image. [#52271](https://github.com/elastic/beats/pull/52271) [#52410](https://github.com/elastic/beats/pull/52410) [#52459](https://github.com/elastic/beats/pull/52459) [#52420](https://github.com/elastic/beats/pull/52420) [#52488](https://github.com/elastic/beats/pull/52488) [#52515](https://github.com/elastic/beats/pull/52515) [#52439](https://github.com/elastic/beats/issues/52439)

  npm 12 no longer runs a dependency&#39;s `install` lifecycle script during `npm i` by
  default, so the transitive playwright-chromium `install` hook that used to download
  the Playwright browsers into the heartbeat image during build stopped running. As a
  result the image shipped with no browsers and all Synthetics browser monitors failed
  with &#34;browserType.launch: Executable doesn&#39;t exist at
  .../chromium_headless_shell-&lt;rev&gt;/...&#34;. The image build now installs the browsers
  explicitly with the bundled Playwright CLI after `npm i`, independent of npm&#39;s
  install-script policy.
  

**Libbeat**

* Prevent stale autodiscover resources after Kubernetes leadership changes. [#52271](https://github.com/elastic/beats/pull/52271) [#52410](https://github.com/elastic/beats/pull/52410) [#52459](https://github.com/elastic/beats/pull/52459) [#52420](https://github.com/elastic/beats/pull/52420) [#52488](https://github.com/elastic/beats/pull/52488) [#52515](https://github.com/elastic/beats/pull/52515) [#52460](https://github.com/elastic/beats/issues/52460)

**Metricbeat**

* Fix Azure module authentication on sovereign clouds (Government, China). [#52271](https://github.com/elastic/beats/pull/52271) [#52410](https://github.com/elastic/beats/pull/52410) [#52459](https://github.com/elastic/beats/pull/52459) [#52420](https://github.com/elastic/beats/pull/52420) [#52488](https://github.com/elastic/beats/pull/52488) [#52515](https://github.com/elastic/beats/pull/52515) [#52460](https://github.com/elastic/beats/issues/52460)

  The Azure module now derives the full cloud configuration (ARM token audience, metrics batch API endpoint and audience) from resource_manager_endpoint. Previously the monitor metricset silently ignored resource_manager_audience, always requesting public-cloud tokens, the billing metricset mutated the SDK&#39;s global cloud configuration, and the metrics batch API endpoint was hardcoded to the public cloud.

