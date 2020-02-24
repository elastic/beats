# uaago

Golang client for UAA.

## Usage

See [samples](/samples).

You can run the auth token sample like this:

```bash
$ go build -o bin/auth_token samples/auth-token/main.go
$ ./bin/auth_token [URL] [USER] [PASS]
```

You can run the refresh token sample like this:

```bash
$ go build -o bin/refresh_token samples/refresh-token/main.go
$ ./bin/refresh_token [URL] [CLIENT_ID] [EXISTING_REFRESH_TOKEN]
```
