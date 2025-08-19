---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-installation-script.html
---

This page applies to:
 - 9.0.6 for versions >= 9.0.0 and < 9.1.0.
 - 9.1.0 for versions >= 9.1.0.

# Installation script
The installation script, `install-service-metricbeat.ps1` is responsible
for creating the Windows Service for Metricbeat. The
base folder has changed from `C:\ProgramData\` to  `C:\Program Files\`
because the latter has stricter permissions, therefore the home path
(base for state and logs) is now `C:\Program Files\Metricbeat-Data`.

The install script (`install-service-metricbeat.ps1`) will check whether
`C:\ProgramData\Metricbeat` exits and attempt to move it to `C:\Program Files\Metricbeat-Data`.
If an error occurs, the script will stop and print the error.

Then it will create the Windows Service setting:
 - `path.home` as `$env:ProgramFiles\Metricbeat-Data`
 - `path.logs` as `$env:ProgramFiles\Metricbeat-Data\logs`

The script also supports passing the parameter `-ForceLegacyPath` to
use the old default `C:\ProgramData\` that is set using
`$env:PROGRAMDATA`. However using `-ForceLegacyPath` is **not
recommended**.

In a PowerShell prompt, can use `Get-Help install-service-metricbeat.ps1
-detailed` to get detailed help.

## Troubleshooting
If there is a permission error when the installation script is moving
the folder, ensure the user running the script has enough permissions
to do so. If the problem persists, the folder can be moved manually,
then the installation script can be executed again.

If the script still cannot move the files, you can manually move
`C:\ProgramData\metricbeat` to `C:\Program Files\Metricbeat-Data`
and run the install script again.
