# Log v2
The LogV2 input is used as an entrypoint to select whether to run the
Log input or (as normally done) the Filestream input as part of
the effort to fully deprecate and remove the Log input.

Currently there are two ways to run the Filestream input in place of
the Log input:
 - When Filebeat is running under Elastic Agent and
   `run_as_filestream: true` is added to the input configuration. This
   allows to control the migration at the input level.
 - When Filebeat is running standalone and the feature flag
   `log_input_run_as_filestream` is enabled (this is done by setting
   `features.log_input_run_as_filestream.enabled: true`). This forces
   *all* Log inputs on Filebeat to run as Filestream.

## Limitations
Regarding of how the migration is enabled, to run the Log input as
Filestream all conditions listed below need to be met: 
 - The Log input configuration must contain an unique ID, this ID will
   become the Filestream input ID.
 - The Container input is not supported
 - Only states of files being actively harvested are migrated.

## Next steps
 - [ ] Support migrating the Container input
 - [ ] [Implement manual fallback mechanism for Filestream running as Log input under Elastic Agent](https://github.com/elastic/beats/issues/47747)
