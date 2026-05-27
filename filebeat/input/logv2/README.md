# Log v2
The LogV2 input is used as an entrypoint to select whether to run the
Log or Container input as Filestream as part of the effort to fully
deprecate and remove the Log input.

The logv2 manager implements the `v2.Redirector` interface. When a
redirect is active the Loader resolves the filestream plugin from its
own registry and calls its `Create` with the translated config. The
logv2 package does not import or instantiate filestream directly.

When no redirect is needed, `Create` returns `v2.ErrUnknownInput` and
`compat.Combine` falls through to the V1 log input as before.

Currently there are two ways to enable the redirect:
 - When Filebeat is running under Elastic Agent and
   `run_as_filestream: true` is added to the input configuration. This
   allows control of the migration at the input level.
 - When Filebeat is running standalone and the feature flag
   `log_input_run_as_filestream` is enabled (this is done by setting
   `features.log_input_run_as_filestream.enabled: true`). This forces
   *all* Log inputs on Filebeat to run as Filestream.

Both the Log and Container input types are supported.

## Limitations
Regardless of how the migration is enabled, to run the Log input as
Filestream all conditions listed below need to be met:
 - The input configuration must contain a unique ID, which becomes the
   Filestream input ID.
 - Only states of files being actively harvested are migrated.

## Next steps
 - [ ] [Implement manual fallback mechanism for Filestream running as Log input under Elastic Agent](https://github.com/elastic/beats/issues/47747)
