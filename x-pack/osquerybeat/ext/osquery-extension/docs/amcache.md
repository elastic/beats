# Amcache

Tables corresponding to keys in the amcache.hve.  Amcache.hve is a registry hive file in Windows that stores information about program executions, installations, and drivers

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Schema

### amcache_application

| Column | Type | Description |
| --- | --- | --- |
| last\_write\_time | BIGINT | Last write time of the application entry. |
| name | TEXT | Application name. |
| program\_id | TEXT | Unique ProgramID from the AmCache. |
| program\_instance\_id | TEXT | Program instance ID. |
| version | TEXT | Application version. |
| publisher | TEXT | Application publisher. |
| language | TEXT | Language code. |
| install\_date | TEXT | Application install date. |
| source | TEXT | Source of the application entry. |
| root\_dir\_path | TEXT | Root directory path of the application. |
| hidden\_arp | TEXT | Whether the program is hidden from Add/Remove Programs. |
| uninstall\_string | TEXT | The application's uninstall string. |
| registry\_key\_path | TEXT | Registry key path for the application. |
| store\_app\_type | TEXT | Type of store app (e.g., UWP). |
| inbox\_modern\_app | TEXT | Whether it is an inbox (pre-installed) modern app. |
| manifest\_path | TEXT | Path to the application manifest. |
| package\_full\_name | TEXT | Full package name. |
| msi\_package\_code | TEXT | MSI package code, if applicable. |
| msi\_product\_code | TEXT | MSI product code, if applicable. |
| msi\_install\_date | TEXT | MSI install date, if applicable. |
| bundle\_manifest\_path | TEXT | Path to the bundle manifest. |
| user\_sid | TEXT | The SID of the user who installed the application. |

### amcache_application_file

| Column | Type | Description |
| --- | --- | --- |
| last\_write\_time | BIGINT | Last write time of the file entry. |
| name | TEXT | File name. |
| program\_id | TEXT | The ProgramID from the AmCache. |
| file\_id | TEXT | The FileID from the AmCache. |
| lower\_case\_long\_path | TEXT | The file's full path, in lowercase. |
| original\_file\_name | TEXT | The original file name from the file's resources. |
| publisher | TEXT | The publisher from the file's resources. |
| version | TEXT | The version from the file's resources. |
| bin\_file\_version | TEXT | Binary file version. |
| binary\_type | TEXT | The binary type (e.g., PE, X86, AMD64). |
| product\_name | TEXT | The product name from the file's resources. |
| product\_version | TEXT | The product version from the file's resources. |
| link\_date | TEXT | The link date from the PE header. |
| bin\_product\_version | TEXT | Binary product version. |
| size | BIGINT | The size of the file in bytes. |
| language | BIGINT | The language code. |
| usn | BIGINT | Update Sequence Number (USN). |
| appx\_package\_full\_name | TEXT | AppX package full name, if applicable. |
| is\_os\_component | TEXT | Whether the file is an OS component. |
| appx\_package\_relative\_id | TEXT | AppX package relative ID, if applicable. |

### amcache_application_shortcut

| Column | Type | Description |
| --- | --- | --- |
| last\_write\_time | BIGINT | Last write time of the shortcut entry. |
| shortcut\_path | TEXT | Full path to the shortcut (.lnk) file. |
| shortcut\_target\_path | TEXT | The target file path the shortcut points to. |
| shortcut\_aumid | TEXT | Application User Model ID (AUMID) for the shortcut. |
| shortcut\_program\_id | TEXT | The ProgramID from AmCache associated with this shortcut. |

### amcache_device_pnp

| Column | Type | Description |
| --- | --- | --- |
| last\_write\_time | BIGINT | Last write time of the device entry. |
| model | TEXT | Device model. |
| manufacturer | TEXT | Device manufacturer. |
| driver\_name | TEXT | Driver file name. |
| parent\_id | TEXT | Device parent ID. |
| matching\_id | TEXT | The matching/hardware ID. |
| class | TEXT | Device class name. |
| class\_guid | TEXT | The device class GUID. |
| description | TEXT | Device description. |
| enumerator | TEXT | Device enumerator (e.g., USB, PCI). |
| service | TEXT | Associated service name. |
| install\_state | TEXT | Device install state. |
| device\_state | TEXT | Device state. |
| inf | TEXT | Name of the INF file. |
| driver\_ver\_date | TEXT | Driver version date. |
| install\_date | TEXT | Device install date. |
| first\_install\_date | TEXT | Device first install date. |
| driver\_package\_strong\_name | TEXT | Strong name of the driver package. |
| driver\_ver\_version | TEXT | Driver version number. |
| container\_id | TEXT | Device container ID. |
| problem\_code | TEXT | Device manager problem code, if any. |
| provider | TEXT | Driver provider. |
| driver\_id | TEXT | Device driver ID. |
| bus\_reported\_description | TEXT | Bus-reported device description. |
| hw\_id | TEXT | Hardware ID. |
| extended\_infs | TEXT | Extended INF files. |
| compid | TEXT | Compatible ID. |
| stack\_id | TEXT | Device stack ID. |
| upper\_class\_filters | TEXT | Upper class filters. |
| lower\_class\_filters | TEXT | Lower class filters. |
| upper\_filters | TEXT | Upper filters. |
G| lower\_filters | TEXT | Lower filters. |
| device\_interface\_classes | TEXT | GUIDs of device interfaces. |
| location\_paths | TEXT | Device location paths. |

### amcache_driver_binary

| Column | Type | Description |
| --- | --- | --- |
| last\_write\_time | BIGINT | Last write time of the driver binary entry. |
| driver\_name | TEXT | File name of the driver binary. |
| inf | TEXT | Name of the INF file. |
| driver\_version | TEXT | Driver version. |
| product | TEXT | Product name from the binary's resources. |
| product\_version | TEXT | Product version from the binary's resources. |
| wdf\_version | TEXT | Windows Driver Framework (WDF) version. |
| driver\_company | TEXT | Company name from the binary's resources. |
| driver\_package\_strong\_name | TEXT | Strong name of the driver package. |
| service | TEXT | Associated service name. |
| driver\_in\_box | TEXT | Whether the driver is included in-box with Windows. |
| driver\_signed | TEXT | Whether the driver is signed. |
| driver\_is\_kernel\_mode | TEXT | Whether the driver is a kernel-mode driver. |
| driver\_id | TEXT | Device driver ID. |

## Examples

### Find Recently Executed Programs
```sql
SELECT
  DATETIME(last_write_time, 'unixepoch') AS last_run_time,
  name,
  lower_case_long_path,
  publisher,
  product_name,
  size
FROM amcache_application_file
ORDER BY last_write_time DESC
LIMIT 100;
```

### Hunt for Suspicious Executable Locations (LOLBAS)

Finds legitimate Windows tools (like powershell.exe) running from non-standard, writable directories.

```sql
SELECT
  name,
  lower_case_long_path,
  publisher,
  DATETIME(last_write_time, 'unixepoch') AS last_run_time
FROM amcache_application_file
WHERE
  name IN (
    'powershell.exe',
    'cmd.exe',
    'cscript.exe',
    'wscript.exe',
    'certutil.exe',
    'bitsadmin.exe',
    'mshta.exe'
  )
  AND lower_case_long_path NOT LIKE 'c:\windows\system32\%'
  AND lower_case_long_path NOT LIKE 'c:\windows\syswow64\%'
  AND lower_case_long_path NOT LIKE 'c:\windows\winsxs\%';
```

### Find Executables with No Publisher or Product Name

Malware and hacking scripts are often compiled without metadata (like a Publisher or Product Name). Legitimate software almost always has this.

```sql
SELECT
  name,
  lower_case_long_path,
  size,
  DATETIME(last_write_time, 'unixepoch') AS last_run_time
FROM amcache_application_file
WHERE
  (publisher IS NULL OR publisher = '')
  AND (product_name IS NULL OR product_name = '')
  AND name LIKE '%.exe'
  AND (
    lower_case_long_path LIKE 'c:\users\%\appdata\local\temp\%'
    OR lower_case_long_path LIKE 'c:\users\public\%'
    OR lower_case_long_path LIKE 'c:\programdata\%'
    OR lower_case_long_path LIKE 'c:\perflogs\%'
    OR lower_case_long_path LIKE 'c:\windows\temp\%'
  )
ORDER BY last_write_time DESC;
```

### Hunt for Known Hacking/Reconnaissance Tool Names

Searches the execution history for file names associated with common attack tools.

```sql
SELECT
  name,
  original_file_name,
  lower_case_long_path,
  publisher,
  DATETIME(last_write_time, 'unixepoch') AS last_run_time
FROM amcache_application_file
WHERE
  name IN (
    'mimikatz.exe',
    'procdump.exe',
    'psexec.exe',
    'nc.exe',
    'ncat.exe',
    'adfind.exe',
    'bloodhound.exe',
    'sharphound.exe',
    'rubeus.exe',
    'seatbelt.exe',
    'plink.exe'
  );
```

### Correlate Installed Apps with Executed Files

Links an installed application to all the individual files it has executed.

```sql
SELECT
  app.name AS application_name,
  app.publisher AS app_publisher,
  app.install_date,
  file.name AS file_name,
  file.lower_case_long_path,
  DATETIME(file.last_write_time, 'unixepoch') AS file_last_run
FROM amcache_application AS app
JOIN amcache_application_file AS file
ON app.program_id = file.program_id

```

### Find Recently Installed Applications

Similar to the "Add/Remove Programs" list, this helps you find recently installed software.

```sql
SELECT
  DATETIME(last_write_time, 'unixepoch') AS last_updated,
  install_date,
  name,
  publisher,
  version,
  uninstall_string
FROM amcache_application
ORDER BY last_write_time DESC
LIMIT 50;
```