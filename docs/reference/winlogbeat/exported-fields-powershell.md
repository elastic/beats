---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/exported-fields-powershell.html
---

# PowerShell module fields [exported-fields-powershell]

These are the event fields specific to the module for the Microsoft-Windows-PowerShell/Operational and Windows PowerShell logs.

**`powershell.id`**
:   Shell Id.

type: keyword

example: Microsoft Powershell


**`powershell.pipeline_id`**
:   Pipeline id.

type: keyword

example: 1


**`powershell.runspace_id`**
:   Runspace id.

type: keyword

example: 4fa9074d-45ab-4e53-9195-e91981ac2bbb


**`powershell.sequence`**
:   Sequence number of the powershell execution.

type: long

example: 1


**`powershell.total`**
:   Total number of messages in the sequence.

type: long

example: 10



## powershell.command [_powershell_command]

Data related to the executed command.

**`powershell.command.path`**
:   Path of the executed command.

type: keyword

example: C:\Windows\system32\cmd.exe


**`powershell.command.name`**
:   Name of the executed command.

type: keyword

example: cmd.exe


**`powershell.command.type`**
:   Type of the executed command.

type: keyword

example: Application


**`powershell.command.value`**
:   The invoked command.

type: text

example: Import-LocalizedData  LocalizedData -filename ArchiveResources


**`powershell.command.invocation_details`**
:   An array of objects containing detailed information of the executed command.

type: array


**`powershell.command.invocation_details.type`**
:   The type of detail.

type: keyword

example: CommandInvocation


**`powershell.command.invocation_details.related_command`**
:   The command to which the detail is related to.

type: keyword

example: Add-Type


**`powershell.command.invocation_details.name`**
:   Only used for ParameterBinding detail type. Indicates the parameter name.

type: keyword

example: AssemblyName


**`powershell.command.invocation_details.value`**
:   The value of the detail. The meaning of it will depend on the detail type.

type: text

example: System.IO.Compression.FileSystem



## powershell.connected_user [_powershell_connected_user]

Data related to the connected user executing the command.

**`powershell.connected_user.domain`**
:   User domain.

type: keyword

example: VAGRANT


**`powershell.connected_user.name`**
:   User name.

type: keyword

example: vagrant



## powershell.engine [_powershell_engine]

Data related to the PowerShell engine.

**`powershell.engine.version`**
:   Version of the PowerShell engine version used to execute the command.

type: keyword

example: 5.1.17763.1007


**`powershell.engine.previous_state`**
:   Previous state of the PowerShell engine.

type: keyword

example: Available


**`powershell.engine.new_state`**
:   New state of the PowerShell engine.

type: keyword

example: Stopped



## powershell.file [_powershell_file]

Data related to the executed script file.

**`powershell.file.script_block_id`**
:   Id of the executed script block.

type: keyword

example: 50d2dbda-7361-4926-a94d-d9eadfdb43fa


**`powershell.file.script_block_text`**
:   Text of the executed script block.

type: text

example: .\a_script.ps1


**`powershell.process.executable_version`**
:   Version of the engine hosting process executable.

type: keyword

example: 5.1.17763.1007



## powershell.provider [_powershell_provider]

Data related to the PowerShell engine host.

**`powershell.provider.new_state`**
:   New state of the PowerShell provider.

type: keyword

example: Active


**`powershell.provider.name`**
:   Provider name.

type: keyword

example: Variable


