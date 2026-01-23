# Summary
## Log Input
- No streams: `filebeat::logs::native::104711-40`
- all: `filebeat::logs::native::104711-40`
- stdout: `filebeat::logs::1b59052b95e61943-native::104711-40`
- stderr: `filebeat::logs::d35e05a633229937-native::104711-40`

The input hashes the `map[string]string` that is passed to `NewInput`
as the `Meta` field from `input.Context`.

## Filestream
- No streams: `filestream::my-id::fingerprint::694805ef26162d16531cb9ea8de6c692e93a0a79fe7e1b331cf456d6d5578750`
- all: `filestream::my-id::fingerprint::694805ef26162d16531cb9ea8de6c692e93a0a79fe7e1b331cf456d6d5578750`
- stdout: `filestream::my-id::fingerprint::694805ef26162d16531cb9ea8de6c692e93a0a79fe7e1b331cf456d6d5578750-stdout`
- stderr: `filestream::my-id::fingerprint::694805ef26162d16531cb9ea8de6c692e93a0a79fe7e1b331cf456d6d5578750-stderr`

The input (via `newFileIdentifier`) adds a suffix to the file identifier. This suffix comes from `config.Reader.Parsers.Suffix`. The `Suffix` field gets set by `NewConfig` (`libbeat/reader/parser/parser.go`)

# Detailed config and registry entry

## Container
### No stream defined
```
  - type: container
    id: stdout-input-id
    allow_deprecated_use: true
    paths:
      -  /tmp/container-log-file.log
```

Key: `filebeat::logs::native::104711-40`
```
{
  "k": "filebeat::logs::native::104711-40",
  "v": {
    "FileStateOS": {
      "device": 40,
      "gid": 1000,
      "inode": 104711,
      "uid": 1000
    },
    "id": "native::104711-40",
    "identifier_name": "native",
    "offset": 16300,
    "prev_id": "",
    "source": "/tmp/container-log-file.log",
    "timestamp": [
      280187435751544,
      1769197808
    ],
    "ttl": -1,
    "type": "container"
  }
}
```
### Stream: all
```
  - type: container
    id: stdout-input-id
    allow_deprecated_use: true
    paths:
      -  /tmp/container-log-file.log
    stream: all
```

Key: `filebeat::logs::native::104711-40`
```
{
  "k": "",
  "v": {
    "FileStateOS": {
      "device": 40,
      "gid": 1000,
      "inode": 104711,
      "uid": 1000
    },
    "id": "native::104711-40",
    "identifier_name": "native",
    "offset": 16300,
    "prev_id": "",
    "source": "/tmp/container-log-file.log",
    "timestamp": [
      280187326690264,
      1769198228
    ],
    "ttl": -1,
    "type": "container"
  }
}
```

### Stream: stdout
```
  - type: container
    id: stdout-input-id
    allow_deprecated_use: true
    paths:
      -  /tmp/container-log-file.log
    stream: stdout
```

Key: `filebeat::logs::1b59052b95e61943-native::104711-40`
```
{
  "k": "filebeat::logs::1b59052b95e61943-native::104711-40",
  "v": {
    "FileStateOS": {
      "device": 40,
      "gid": 1000,
      "inode": 104711,
      "uid": 1000
    },
    "id": "1b59052b95e61943-native::104711-40",
    "identifier_name": "native",
    "meta": {
      "stream": "stdout"
    },
    "offset": 16218,
    "prev_id": "",
    "source": "/tmp/container-log-file.log",
    "timestamp": [
      280186823624425,
      1769198308
    ],
    "ttl": -1,
    "type": "container"
  }
}
```

### Stream: stderr
```
  - type: container
    id: stdout-input-id
    allow_deprecated_use: true
    paths:
      -  /tmp/container-log-file.log
    stream: stderr
```

Key: `filebeat::logs::d35e05a633229937-native::104711-40`
```
{
  "k": "filebeat::logs::d35e05a633229937-native::104711-40",
  "v": {
    "FileStateOS": {
      "device": 40,
      "gid": 1000,
      "inode": 104711,
      "uid": 1000
    },
    "id": "d35e05a633229937-native::104711-40",
    "identifier_name": "native",
    "meta": {
      "stream": "stderr"
    },
    "offset": 16300,
    "prev_id": "",
    "source": "/tmp/container-log-file.log",
    "timestamp": [
      280186580081410,
      1769198413
    ],
    "ttl": -1,
    "type": "container"
  }
}
```

## Filestream
### Stream not defined
```
  - type: filestream
    id: my-id
    paths:
      -  /tmp/container-log-file.log    
    parsers:
      - container: ~
```

Key: `filestream::my-id::fingerprint::694805ef26162d16531cb9ea8de6c692e93a0a79fe7e1b331cf456d6d5578750`
```
{
  "k": "filestream::my-id::fingerprint::694805ef26162d16531cb9ea8de6c692e93a0a79fe7e1b331cf456d6d5578750",
  "v": {
    "cursor": {
      "eof": false,
      "offset": 16300
    },
    "meta": {
      "identifier_name": "fingerprint",
      "source": "/tmp/container-log-file.log"
    },
    "ttl": -1,
    "updated": [
      280187083233340,
      1769198703
    ]
  }
}
```

### Stream: all
```
  - type: filestream
    id: my-id
    paths:
      -  /tmp/container-log-file.log    
    parsers:
      - container:
          stream: all
```

Key: `filestream::my-id::fingerprint::694805ef26162d16531cb9ea8de6c692e93a0a79fe7e1b331cf456d6d5578750`
```
{
  "k": "filestream::my-id::fingerprint::694805ef26162d16531cb9ea8de6c692e93a0a79fe7e1b331cf456d6d5578750",
  "v": {
    "cursor": {
      "eof": false,
      "offset": 16300
    },
    "meta": {
      "identifier_name": "fingerprint",
      "source": "/tmp/container-log-file.log"
    },
    "ttl": -1,
    "updated": [
      280187194679346,
      1769198811
    ]
  }
}
```

### Stream: stdout
```
  - type: filestream
    id: my-id
    paths:
      -  /tmp/container-log-file.log    
    parsers:
      - container:
          stream: stdout
```

Key: `filestream::my-id::fingerprint::694805ef26162d16531cb9ea8de6c692e93a0a79fe7e1b331cf456d6d5578750-stdout`
```
{
  "k": "filestream::my-id::fingerprint::694805ef26162d16531cb9ea8de6c692e93a0a79fe7e1b331cf456d6d5578750-stdout",
  "v": {
    "cursor": {
      "eof": false,
      "offset": 16218
    },
    "meta": {
      "identifier_name": "fingerprint",
      "source": "/tmp/container-log-file.log"
    },
    "ttl": -1,
    "updated": [
      280187157676158,
      1769198919
    ]
  }
}
```

### Stream: stderr
```
  - type: filestream
    id: my-id
    paths:
      -  /tmp/container-log-file.log    
    parsers:
      - container:
          stream: stderr
```

Key: `filestream::my-id::fingerprint::694805ef26162d16531cb9ea8de6c692e93a0a79fe7e1b331cf456d6d5578750-stderr`
```
{
  "k": "filestream::my-id::fingerprint::694805ef26162d16531cb9ea8de6c692e93a0a79fe7e1b331cf456d6d5578750-stderr",
  "v": {
    "cursor": {
      "eof": false,
      "offset": 16300
    },
    "meta": {
      "identifier_name": "fingerprint",
      "source": "/tmp/container-log-file.log"
    },
    "ttl": -1,
    "updated": [
      280186809086134,
      1769199093
    ]
  }
}
```
