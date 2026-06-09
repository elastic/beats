---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/auditbeat-installation-script.html
applies_to:
  stack: ga 9.0.6
---

# Installation script
The installation script, `install-service-auditbeat.ps1` is responsible
for creating the Windows Service for Auditbeat. Starting in 9.0.6, the
base folder has changed from `C:\ProgramData\` to  `C:\Program Files\`
because the latter has stricter permissions, therefore the home path
(base for state and logs) is now `C:\Program Files\Auditbeat-Data`.

The install script (`install-service-auditbeat.ps1`) will check whether
`C:\ProgramData\Auditbeat` exits and attempt to move it to `C:\Program Files\Auditbeat-Data`.
If an error occurs, the script will stop and print the error.

Then it will create the Windows Service setting:
 - `path.home` as `$env:ProgramFiles\Auditbeat-Data`
 - `path.logs` as `$env:ProgramFiles\Auditbeat-Data\logs`

The script also supports passing the parameter `-ForceLegacyPath` to
use the old default `C:\ProgramData\` that is set using
`$env:PROGRAMDATA`. However using `-ForceLegacyPath` is **not
recommended**.

In a PowerShell prompt, can use `Get-Help install-service-auditbeat.ps1
-detailed` to get detailed help.

## Troubleshooting
If there is a permission error when the installation script is moving
the folder, ensure the user running the script has enough permissions
to do so. If the problem persists, the folder can be moved manually,
then the installation script can be executed again.
