`tablespace` Metricset includes information about data files and temp files, grouped by Tablespace with free space available, used space, status of the data files, status of the Tablespace, etc.


## Required database access [_required_database_access_3]

To ensure that the module has access to the appropriate metrics, the module requires that you configure a user with access to the following tables:

* SYS.DBA_TEMP_FILES
* DBA_TEMP_FREE_SPACE
* dba_data_files
* dba_free_space


## Description of fields [_description_of_fields_2]

* **data_file.id**: Tablespace data file unique identifier number. Each data file of a Tablespace has a unique name (and each Tablespace may have more than one data file) but this is not the Tablespace ID.
* **data_file.name**: Filename of the data file (with the full path)
* **data_file.online_status**: Last known online status of the data file. One of SYSOFF, SYSTEM, OFFLINE, ONLINE or RECOVER.
* **data_file.size.bytes**: Size of the file in bytes.
* **data_file.size.free.bytes**: The size of the file available for user data. The actual size of the file minus this value is used to store file related metadata.
* **data_file.size.max.bytes**: Maximum file size in bytes
* **data_file.status**: File status: AVAILABLE or INVALID (INVALID means that the file number is not in use, for example, a file in a tablespace that was dropped)
* **name**: Tablespace name
* **space.free.bytes**: Tablespace total free space available, in bytes.
* **space.total.bytes**: Tablespace total size, in bytes. Calculated by adding the file sizes for each Tablespace.
* **space.used.bytes**: Tablespace used space, in bytes.
