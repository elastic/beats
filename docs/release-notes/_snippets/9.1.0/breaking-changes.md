## 9.1.0 [beats-9.1.0-breaking-changes]

**All Beats**

::::{dropdown} The default data and logs path for the Windows service installation has changed.
The base folder has changed from `C:\ProgramData\` to `C:\ProgramFiles\`
because the latter has stricter permissions. The state
and logs are now stored in `C:\Program Files\<Beat Name>-Data`.

When the installation script runs, it looks for the previous default
data path. If the path is found, data is moved to the new path.
The installation script accepts the parameter `-ForceLegacyPath` to
force using the legacy data path.

In a PowerShell prompt, use `Get-Help install-service-<Beat Name>.ps1
-detailed` to get detailed help.

See 'Quick start -> Installation script' from each Beat for more
details.

::::

**Filebeat**

::::{dropdown} 'close.on_state_change.removed' defaults to `true` on Windows and `false` on the rest of the platforms.
To keep the previous behaviour, add `close.on_state_change.removed:
true` on every Filestream input.

Even after the file is removed, the file handles will stay open until
it is closed due to
inactivity. See [`close.on_state_change.inactive`](https://www.elastic.co/docs/reference/beats/filebeat/filebeat-input-filestream#filebeat-input-filestream-close-inactive)
for more details.

For more information, check [#38523](https://github.com/elastic/beats/issues/38523)
::::
