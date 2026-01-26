## 9.0.6 [beats-9.0.6-breaking-changes]

**All Beats**

::::{dropdown} The default data and logs path for the Windows service installation has changed.
The base folder has changed from `C:\ProgramData\` to
`C:\ProgramFiles\` because the latter has stricter permissions. The state
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