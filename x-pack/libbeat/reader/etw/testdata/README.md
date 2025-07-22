# ETW Test Data and Tooling

This directory contains the manifest, tools, and test data for the Beats ETW reader.

**Prerequisite**: All commands including tests must be run from a terminal with **Administrator privileges**.

-----

## Manifest Compilation

The `sample.dll` file is pre-compiled and included in the repository. You only need to recompile it if you modify the `sample.man` manifest file.

**To recompile:**

  * Requires the Windows SDK or Visual Studio Build Tools.
  * Run the `compile-manifest.ps1` script from the `testdata` directory.

```powershell
# From the testdata directory
.\compile-manifest.ps1
```

-----

## Test Data Regeneration

The test suite uses an ETL trace file and a JSON golden file for validation.

### Regenerate ETL File

To generate a new `sample-test-events.etl` file, run the `TestRegenerateTestdataETL` test with the `-regenerate-etl` flag.

```bash
# From the parent `etw` directory
go test -v -run TestRegenerateTestdataETL -regenerate-etl
```

### Regenerate Golden File

To update `golden_events.json` with new parser output, run the `TestETLGoldenFile` test with the `-regenerate-golden` flag.

```bash
# From the parent `etw` directory
go test -v -run TestETLGoldenFile -regenerate-golden
```

-----

## Running Benchmarks

To measure the performance of the ETW reader, run the `BenchmarkETWCallbackRate` benchmark.

```bash
# From the parent `etw` directory
go test -bench=BenchmarkETWCallbackRate -benchmem -v
```