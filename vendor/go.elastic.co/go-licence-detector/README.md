Go Licence Detector
===================

This is a tool designed to generate licence notices and dependency listings for Go projects at Elastic. It parses the output of `go list -m -json all` to produce its output.

```
go get go.elastic.co/go-licence-detector
```

## Usage

```
go-licence-detector [FLAGS]

Flags:
  -depsOut string
    	Path to output the dependency list.
  -depsTemplate string
    	Path to the dependency list template file. (default "example/templates/dependencies.asciidoc.tmpl")
  -in string
    	Dependency list (output from go list -m -json all). (default "-")
  -includeIndirect
    	Include indirect dependencies.
  -licenceData string
    	Path to the licence database. Uses embedded database if empty.
  -noticeOut string
    	Path to output the notice.
  -noticeTemplate string
    	Path to the NOTICE template file. (default "example/templates/NOTICE.txt.tmpl")
  -overrides string
    	Path to the file containing override directives.
  -rules string
    	Path to file containing rules regarding licence types. Uses embedded rules if empty.
  -validate
    	Validate results (slow).

Example:
   $ go list -m -json all | go-licence-detector -includeIndirect -depsOut=dependencies.asciidoc -noticeOut=NOTICE.txt
```

If no file path is provided for `-noticeOut` or `-depsOut`, the corresponding output will not be generated. 


## Adding rules

Allowed licence types can be specified using a JSON file with the following structure:

```json
{
  "allowlist": [
    "Apache-2.0",
    "MIT"
  ]
}
```

A partial list of allowed licences at Elastic is included in `assets/rules.json` and used by default if no other rules file is specified using the `-rules` flag.


## Adding overrides

In some cases, the application will not be able to detect the licence type or infer the correct URL for a dependency. When there are issues with licences (no licence file or unknown licence type), the application will fail with an error message instructing the user to add an override to continue. The overrides file is a file containing newline-delimited JSON where each line contains a JSON object bearing the following format:

- `name`: Required. Module name to apply the override to.
- `licenceFile`: Optional. Path to a file containing the licence text for this module under the module directory. It must be relative to the dependency path.
- `licenceType`: Optional. Type of licence (Apache-2.0, ISC etc.). Provide a [SPDX](https://spdx.org/licenses/) identifier.
- `licenceTextOverrideFile`: Optional. Path to a file containing the licence text for this module. Path must be relative to the `overrides.json` file.
- `url`: Optional. URL to the dependency website.

Example overrides file:

```json
{"name": "github.com/bmizerany/perks", "licenceTextOverrideFile": "licences/github.com/bmizerany/perks/LICENCE"}
{"name": "github.com/dgryski/go-gk", "licenceType": "MIT"}
{"name": "github.com/russross/blackfriday/v2", "url": "https://gopkg.in/russross/blackfriday.v2"}
```

See `example/overrides` for the suggested structure of adding overrides.


## Validating URLs

Dependency URLs are inferred from the module path. In some rare cases, these URLs could be invalid. Passing the `-validate` flag will make the licence-detector attempt to validate each URL it detects. Please note that this process makes network requests to each of the detected URLs. Running this step in an automated fashion (such as a CI environment) is not recommended.


## Updating the licence database

The licence database file `licence.db` contains all the currently known licence types found in https://github.com/google/licenseclassifier/tree/master/licenses. In the rare case that entirely new licence types have been introduced to the codebase, follow the instructions at https://github.com/google/licenseclassifier to execute the `license_serializer` tool.
