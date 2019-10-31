[![Build Status](https://travis-ci.org/go-goracle/goracle.svg?branch=v2)](https://travis-ci.org/go-goracle/goracle)
[![GoDoc](https://godoc.org/gopkg.in/goracle.v2?status.svg)](http://godoc.org/gopkg.in/goracle.v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-goracle/goracle)](https://goreportcard.com/report/github.com/go-goracle/goracle)
[![codecov](https://codecov.io/gh/go-goracle/goracle/branch/master/graph/badge.svg)](https://codecov.io/gh/go-goracle/goracle)

# goracle

[goracle](driver.go) is a package which is a
[database/sql/driver.Driver](http://golang.org/pkg/database/sql/driver/#Driver)
for connecting to Oracle DB, using Anthony Tuininga's excellent OCI wrapper,
[ODPI-C](https://www.github.com/oracle/odpi).

At least Go 1.11 is required!

## Connect

In `sql.Open("goracle", connString)`, you can provide the classic "user/passw@service_name"
as connString, or an URL like "oracle://user:passw@service_name".

You can provide all possible options with `ConnectionParams`.
Watch out the `ConnectionParams.String()` does redact the password
(for security, to avoid logging it - see https://github.com/go-goracle/goracle/issues/79).
So use `ConnectionParams.StringWithPassword()`.

More advanced configurations can be set with a connection string such as:
`user/pass@(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=tcp)(HOST=hostname)(PORT=port)))(CONNECT_DATA=(SERVICE_NAME=sn)))`

A configuration like this is how you would add functionality such as load balancing across mutliple servers. The portion
described in parenthesis above can also be set in the `SID` field of `ConnectionParams`.

For other possible connection strings, see https://oracle.github.io/node-oracledb/doc/api.html#connectionstrings
and https://docs.oracle.com/en/database/oracle/oracle-database/12.2/netag/configuring-naming-methods.html#GUID-B0437826-43C1-49EC-A94D-B650B6A4A6EE .

TL;DR; the short form is `username@[//]host[:port][/service_name][:server][/instance_name]`, the long form is
`(DESCRIPTION= (ADDRESS=(PROTOCOL=tcp)(HOST=host)(PORT=port)) (CONNECT_DATA= (SERVICE_NAME=service_name) (SERVER=server) (INSTANCE_NAME=instance_name)))`.

To use heterogeneous pools, set `heterogeneousPool=1` and provide the username/password through
`goracle.ContextWithUserPassw`.

## Rationale

With Go 1.9, driver-specific things are not needed, everything (I need) can be
achieved with the standard _database/sql_ library. Even calling stored procedures
with OUT parameters, or sending/retrieving PL/SQL array types - just give a
`goracle.PlSQLArrays` Option within the parameters of `Exec`!

The array size of the returned PL/SQL arrays can be set with `goracle.ArraySize(2000)`

* the default is 1024.

Connections are pooled by default (except `AS SYSOPER` or `AS SYSDBA`).

## Speed

Correctness and simplicity is more important than speed, but the underlying ODPI-C library
helps a lot with the lower levels, so the performance is not bad.

Queries are prefetched (256 rows by default, can be changed by adding a
`goracle.FetchRowCount(1000)` argument to the call of Query),
but you can speed up INSERT/UPDATE/DELETE statements
by providing all the subsequent parameters at once, by putting each param's subsequent
elements in a separate slice:

Instead of

    db.Exec("INSERT INTO table (a, b) VALUES (:1, :2)", 1, "a")
    db.Exec("INSERT INTO table (a, b) VALUES (:1, :2)", 2, "b")

do

    db.Exec("INSERT INTO table (a, b) VALUES (:1, :2)", []int{1, 2}, []string{"a", "b"})

## Logging

Goracle uses `github.com/go-kit/kit/log`'s concept of a `Log` function.
Either set `goracle.Log` to a logging function globally,
or (better) set the logger in the Context of ExecContext or QueryContext:

    db.QueryContext(goracle.ContextWithLog(ctx, logger.Log), qry)

## Tracing

To set ClientIdentifier, ClientInfo, Module, Action and DbOp on the session,
to be seen in the Database by the Admin, set goracle.TraceTag on the Context:

    db.QueryContext(goracle.ContextWithTraceTag(goracle.TraceTag{
    	Module: "processing",
    	Action: "first",
    }), qry)

## Extras

To use the goracle-specific functions, you'll need a `*goracle.conn`.
That's what `goracle.DriverConn` is for!
See [z_qrcn_test.go](./z_qrcn_test.go) for using that to reach
[NewSubscription](https://godoc.org/gopkg.in/goracle.v2#Subscription).

### Calling stored procedures
Use `ExecContext` and mark each OUT parameter with `sql.Out`.

### Using cursors returned by stored procedures
Use `ExecContext` and an `interface{}` or a `database/sql/driver.Rows` as the `sql.Out` destination,
then either use the `driver.Rows` interface,
or transform it into a regular `*sql.Rows` with `goracle.WrapRows`,
or (since Go 1.12) just Scan into `*sql.Rows`.

For examples, see Anthony Tuininga's
[presentation about Go](https://static.rainfocus.com/oracle/oow18/sess/1525791357522001Q5tc/PF/DEV5047%20-%20The%20Go%20Language_1540587475596001afdk.pdf)
(page 39)!


## Caveats

### sql.NullString

`sql.NullString` is not supported: Oracle DB does not differentiate between
an empty string ("") and a NULL, so an

    sql.NullString{String:"", Valid:true} == sql.NullString{String:"", Valid:false}

and this would be more confusing than not supporting `sql.NullString` at all.

Just use plain old `string` !

### NUMBER

`NUMBER`s are transferred as `goracle.Number` (which is a `string`) to Go under the hood.
This ensures that we don't lose any precision (Oracle's NUMBER has 38 decimal digits),
and `sql.Scan` will hide this and `Scan` into your `int64`, `float64` or `string`, as you wish.

For `PLS_INTEGER` and `BINARY_INTEGER` (PL/SQL data types) you can use `int32`.

### CLOB, BLOB
From 2.9.0, LOBs are returned as string/[]byte by default (before it needed the `ClobAsString()` option).
Now it's reversed, and the default is string, to get a Lob reader, give the `LobAsReader()` option.

If you return Lob as a reader, watch out with `sql.QueryRow`, `sql.QueryRowContext` !
They close the statement right after you `Scan` from the returned `*Row`, the returned `Lob` will be invalid, producing
`getSize: ORA-00000: DPI-1002: invalid dpiLob handle`.

So, use a separate `Stmt` or `sql.QueryContext`.

For writing a LOB, the LOB locator returned from the database is valid only till the `Stmt` is valid!
So `Prepare` the statement for the retrieval, then `Exec`, and only `Close` the stmt iff you've finished with your LOB!
For example, see [z_lob_test.go](./z_lob_test.go), `TestLOBAppend`.

### TIMESTAMP
As I couldn't make TIMESTAMP arrays work, all `time.Time` is bind as `DATE`, so fractional seconds
are lost.
A workaround is converting to string:

    time.Now().Format("2-Jan-06 3:04:05.000000 PM")

See #121.


# Install

Just

    go get gopkg.in/goracle.v2

Or if you prefer `dep`

    dep ensure -add gopkg.in/goracle.v2

and you're ready to go!

Note that Windows may need some newer gcc (mingw-w64 with gcc 7.2.0).

## Contribute

Just as with other Go projects, you don't want to change the import paths, but you can hack on the library
in place, just set up different remotes:

    cd $GOPATH.src/gopkg.in/goracle.v2
    git remote add upstream https://github.com/go-goracle/goracle.git
    git fetch upstream
    git checkout -b master upstream/master

    git checkout -f master
    git pull upstream master
    git remote add fork git@github.com:mygithubacc/goracle
    git checkout -b newfeature upstream/master

Change, experiment as you wish, then

    git commit -m 'my great changes' *.go
    git push fork newfeature

and you're ready to send a GitHub Pull Request from `github.com/mygithubacc/goracle`, `newfeature` branch.

### pre-commit

Add this to .git/hooks/pre-commit (after `go get github.com/golangci/golangci-lint/cmd/golangci-lint`)

```
#!/bin/sh
set -e

output="$(gofmt -l "$@")"

if [ -n "$output" ]; then
	echo >&2 "Go files must be formatted with gofmt. Please run:"
	for f in $output; do
		echo >&2 "  gofmt -w $PWD/$f"
	done
	exit 1
fi

golangci-lint run
```

# Third-party

* [oracall](https://github.com/tgulacsi/oracall) generates a server for calling stored procedures.
