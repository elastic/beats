# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.15.3] - 2019-05-16
### Changed
- ParseConnString: reorder logic to allow 'sys/... as sysdba' (without @)

## [2.15.3] - 2019-05-16
### Changed
- ParseConnString: reorder logic to allow 'sys/... as sysdba' (without @)

## [2.15.2] - 2019-05-12
### Changed
- Use time.Local if it equals with DBTIMEZONE (use DST of time.Local).

## [2.15.1] - 2019-05-09
### Changed
- Fix heterogenous pools (broken with 2.14.1)

## [2.15.0] - 2019-05-09
### Added
- Implement dataGetObject to access custom user types
- Add ObjectScanner and ObjectWriter interfaces to provide a way to load/update values from/to a struct and database object type.

## [2.14.2] - 2019-05-07
### Added
- Cache timezone with the pool and in the conn struct, too.

## [2.14.1] - 2019-05-07
- Try to get the serve DBTIMEZONE, if fails use time.Local

## [2.14.0] - 2019-05-07
### Changed
- Default to time.Local in DATE types when sending to DB, too.

## [2.13.2] - 2019-05-07
### Changed
- Default to time.Local timezone for DATE types.

## [2.13.1] - 2019-05-06
### Changed
- Fix 'INTERVAL DAY TO SECOND' NULL case.

## [2.12.8] - 2019-05-02
### Added
- NewConnector, NewSessionIniter

## [2.12.7] - 2019-04-24
### Changed
- ODPI-C v3.1.4 (rowcount for PL/SQL block)

## [2.12.6] - 2019-04-12
### Added
- Allow calling with LOB got from DB, and don't copy it - see #135.

## [2.12.5] - 2019-04-03
### Added
- Make it compile under Go 1.9.

## [2.12.4] - 2019-03-13
## Added
- Upgrade to ODPI-C v3.1.3

## [2.12.3] - 2019-02-20
### Changed
- Use ODPI-C v3.1.1
### Added
- Make goracle.drv implement driver.DriverContext with OpenConnector.

## [2.12.2] - 2019-02-15
### Changed
- Use ODPI-C v3.1.1

## [2.12.0] - 2019-01-21
### Changed
- Use ODPI-C v3.1.0

## [2.11.2] - 2019-01-15
### Changed
- ISOLATION LEVEL READ COMMITTED (typo) fix.

## [2.11.1] - 2018-12-13
### Changed
- Use C.dpiAuthMode, C.dpiStartupMode, C.dpiShutdownMode instead of C.uint - for #129.

## [2.11.0] - 2018-12-13
### Changed
- Do not set empty SID from ORACLE_SID/TWO_TASK environment variables, leave it to ODPI.

### Added
- Allow PRELIM authentication to allow Startup and Shutdown.

## [2.10.1] - 2018-11-23
### Changed
- Don't call SET TRANSACTION if not really needed in BeginTx - if the isolation level hasn't changed.

## [2.10.0] - 2018-11-18
### Added
- Implement RowsNextResultSet to return implicit result sets set by DBMS_SQL.return.
- Allow using heterogeneous pools with user set with ContextWithUserPassw.

## [2.9.1] - 2018-11-14
### Added
- allow RETURNING with empty result set (such as UPDATE).
- Allow SELECT to return object types.

### Changed
- fixed Number.MarshalJSON (see #112)'

## [2.9.0] - 2018-10-12
### Changed
- The default type for BLOB is []byte and for CLOB is a string - no need for ClobAsString() option.

## [2.8.2] - 2018-10-01
### Changed
- Fix the driver.Valuer handling, make it the last resort

## [2.8.1] - 2018-09-27
### Added
- CallTimeout option to set a per-statement OCI_ATTR_CALL_TIMEOUT.
- Allow login with " AS SYSASM", as requested in #100.

### Changed
- Hash the password ("SECRET-sasdas=") in ConnectionParams.String().

## [2.8.0] - 2018-09-21
### Added
- WrapRows wraps a driver.Rows (such as a returned cursor from a stored procedure) as an sql.Rows for easier handling.

### Changed
- Do not allow events by default, make them opt-in with EnableEvents connection parameter - see #98.

## [2.7.1] - 2018-09-17
### Changed
- Inherit parent statement's Options for statements returned as sql.Out.

## [2.7.0] - 2018-09-14
### Changed
- Update ODPI-C to v3.0.0.

## [2.6.0] - 2018-08-31
### Changed
- convert named types to their underlying scalar values - see #96, using MagicTypeConversion() option.

## [2.5.11] - 2018-08-30
### Added
- Allow driver.Valuer as Query argument - see #94.

## [2.5.10] - 2018-08-26
### Changed
- use sergeymakinen/oracle-instant-client:12.2 docker for tests
- added ODPI-C and other licenses into LICENSE.md
- fill varInfo.ObjectType for better Object support

## [2.5.9] - 2018-08-03
### Added
- add CHANGELOG
- check that `len(dest) == len(rows.columns)` in `rows.Next(dest)`

### Changed
- after a Break, don't release a stmt, that may fail with SIGSEGV - see #84.

## [2.5.8] - 2018-07-27
### Changed
- noConnectionPooling option became standaloneConnection

## [2.5.7] - 2018-07-25
### Added
- noConnectionPooling option to force not using a session pool

## [2.5.6] - 2018-07-18
### Changed
- use ODPI-C v2.4.2
- remove all logging/printing of passwords

## [2.5.5] - 2018-07-03
### Added
- allow *int with nil value to be used as NULL

## [2.5.4] - 2018-06-29
### Added
- allow ReadOnly transactions

## [2.5.3] - 2018-06-29
### Changed
- decrease maxArraySize to be compilable on 32-bit architectures.

### Removed
- remove C struct size Printf

## [2.5.2] - 2018-06-22
### Changed
- fix liveness check in statement.Close

## [2.5.1] - 2018-06-15
### Changed
- sid -> service_name in docs
- travis: 1.10.3
- less embedding of structs, clearer API docs

### Added
- support RETURNING from DML
- set timeouts on poolCreateParams

## [2.5.0] - 2018-05-15
### Changed
- update ODPI-C to v2.4.0
- initialize context / load lib only on first Open, to allow import without Oracle Client installed
- use golangci-lint


