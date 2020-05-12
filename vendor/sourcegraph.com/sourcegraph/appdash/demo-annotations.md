# Demo Annotations

## Important

If you're not looking to understand the internal of Appdash, you're probably in the wrong place.

This document gives a large-scale overview of the collections that occur (i.e. would be sent to a remote Go/Appdash collection server) when running the `cmd/appdash demo` demo and navigating to `http://localhost:8699/api-calls` in Google Chrome.

## Span Format

Remember that the span format is `trace-id/span-id/parent-id`, so for example `8d4bdb285382e850/3be70fd9713c47cf/d9564f71aae8aba2` -- `8d4` is the trace/root ID, `3be` is the span's ID, and `d95` is the parent span's ID.

### Collection 1
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/3be70fd9713c47cf/d9564f71aae8aba2"
    len(anns) = 2

Annotation(key="Name", value="localhost:8699")
Annotation(key="_schema:name", value="")
```

### Collection 2
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/3be70fd9713c47cf/d9564f71aae8aba2"
    len(anns) = 2

Annotation(key="Name", value="/endpoint-A")
Annotation(key="_schema:name", value="")
```

### Collection 3
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/3be70fd9713c47cf/d9564f71aae8aba2"
    len(anns) = 17

Annotation(key="ServerRecv", value="2015-02-21T17:02:28.739872792-07:00")
Annotation(key="ServerSend", value="2015-02-21T17:02:28.990097629-07:00")
Annotation(key="Request.ContentLength", value="0")
Annotation(key="Request.Method", value="GET")
Annotation(key="Request.URI", value="/endpoint-A")
Annotation(key="Request.Proto", value="HTTP/1.1")
Annotation(key="Request.Headers.User-Agent", value="Go 1.1 package http")
Annotation(key="Request.Headers.Span-Id", value="8d4bdb285382e850/3be70fd9713c47cf/d9564f71aae8aba2")
Annotation(key="Request.Headers.Accept-Encoding", value="gzip")
Annotation(key="Request.Host", value="localhost:8699")
Annotation(key="Request.RemoteAddr", value="127.0.0.1:35786")
Annotation(key="Response.Headers.Span-Id", value="8d4bdb285382e850/3be70fd9713c47cf/d9564f71aae8aba2")
Annotation(key="Response.ContentLength", value="23")
Annotation(key="Response.StatusCode", value="200")
Annotation(key="Route", value="/endpoint-A")
Annotation(key="User", value="")
Annotation(key="_schema:HTTPServer", value="")
```

### Collection 4
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/3be70fd9713c47cf/d9564f71aae8aba2"
    len(anns) = 15

Annotation(key="Response.StatusCode", value="200")
Annotation(key="Response.Headers.Date", value="Sun, 22 Feb 2015 00:02:28 GMT")
Annotation(key="Response.Headers.Content-Length", value="23")
Annotation(key="Response.Headers.Content-Type", value="text/plain; charset=utf-8")
Annotation(key="Response.ContentLength", value="23")
Annotation(key="ClientSend", value="2015-02-21T17:02:28.738020881-07:00")
Annotation(key="ClientRecv", value="2015-02-21T17:02:28.990831661-07:00")
Annotation(key="Request.Method", value="GET")
Annotation(key="Request.URI", value="/endpoint-A")
Annotation(key="Request.Proto", value="HTTP/1.1")
Annotation(key="Request.Headers.Span-Id", value="8d4bdb285382e850/3be70fd9713c47cf/d9564f71aae8aba2")
Annotation(key="Request.Host", value="localhost:8699")
Annotation(key="Request.RemoteAddr", value="")
Annotation(key="Request.ContentLength", value="0")
Annotation(key="_schema:HTTPClient", value="")
```

### Collection 5
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/ef1ba6f3fa12d5d2/d9564f71aae8aba2"
    len(anns) = 2

Annotation(key="Name", value="localhost:8699")
Annotation(key="_schema:name", value="")
```

### Collection 6
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/ef1ba6f3fa12d5d2/d9564f71aae8aba2"
    len(anns) = 2

Annotation(key="Name", value="/endpoint-B")
Annotation(key="_schema:name", value="")
```

### Collection 7
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/ef1ba6f3fa12d5d2/d9564f71aae8aba2"
    len(anns) = 17

Annotation(key="Response.StatusCode", value="200")
Annotation(key="Response.Headers.Span-Id", value="8d4bdb285382e850/ef1ba6f3fa12d5d2/d9564f71aae8aba2")
Annotation(key="Response.ContentLength", value="28")
Annotation(key="Route", value="/endpoint-B")
Annotation(key="User", value="")
Annotation(key="ServerRecv", value="2015-02-21T17:02:28.991944372-07:00")
Annotation(key="ServerSend", value="2015-02-21T17:02:29.067157987-07:00")
Annotation(key="Request.URI", value="/endpoint-B")
Annotation(key="Request.Proto", value="HTTP/1.1")
Annotation(key="Request.Headers.Span-Id", value="8d4bdb285382e850/ef1ba6f3fa12d5d2/d9564f71aae8aba2")
Annotation(key="Request.Headers.Accept-Encoding", value="gzip")
Annotation(key="Request.Headers.User-Agent", value="Go 1.1 package http")
Annotation(key="Request.Host", value="localhost:8699")
Annotation(key="Request.RemoteAddr", value="127.0.0.1:35787")
Annotation(key="Request.ContentLength", value="0")
Annotation(key="Request.Method", value="GET")
Annotation(key="_schema:HTTPServer", value="")
```

### Collection 8
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/ef1ba6f3fa12d5d2/d9564f71aae8aba2"
    len(anns) = 15

Annotation(key="Request.Method", value="GET")
Annotation(key="Request.URI", value="/endpoint-B")
Annotation(key="Request.Proto", value="HTTP/1.1")
Annotation(key="Request.Headers.Span-Id", value="8d4bdb285382e850/ef1ba6f3fa12d5d2/d9564f71aae8aba2")
Annotation(key="Request.Host", value="localhost:8699")
Annotation(key="Request.RemoteAddr", value="")
Annotation(key="Request.ContentLength", value="0")
Annotation(key="Response.StatusCode", value="200")
Annotation(key="Response.Headers.Date", value="Sun, 22 Feb 2015 00:02:29 GMT")
Annotation(key="Response.Headers.Content-Length", value="28")
Annotation(key="Response.Headers.Content-Type", value="text/plain; charset=utf-8")
Annotation(key="Response.ContentLength", value="28")
Annotation(key="ClientSend", value="2015-02-21T17:02:28.991155375-07:00")
Annotation(key="ClientRecv", value="2015-02-21T17:02:29.067694228-07:00")
Annotation(key="_schema:HTTPClient", value="")
```

### Collection 9
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/910810f4ade66b9d/d9564f71aae8aba2"
    len(anns) = 2

Annotation(key="Name", value="localhost:8699")
Annotation(key="_schema:name", value="")
```

### Collection 10
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/910810f4ade66b9d/d9564f71aae8aba2"
    len(anns) = 2

Annotation(key="Name", value="/endpoint-C")
Annotation(key="_schema:name", value="")
```

### Collection 11
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/910810f4ade66b9d/d9564f71aae8aba2"
    len(anns) = 17

Annotation(key="ServerRecv", value="2015-02-21T17:02:29.068944038-07:00")
Annotation(key="ServerSend", value="2015-02-21T17:02:29.369228802-07:00")
Annotation(key="Request.Method", value="GET")
Annotation(key="Request.URI", value="/endpoint-C")
Annotation(key="Request.Proto", value="HTTP/1.1")
Annotation(key="Request.Headers.User-Agent", value="Go 1.1 package http")
Annotation(key="Request.Headers.Span-Id", value="8d4bdb285382e850/910810f4ade66b9d/d9564f71aae8aba2")
Annotation(key="Request.Headers.Accept-Encoding", value="gzip")
Annotation(key="Request.Host", value="localhost:8699")
Annotation(key="Request.RemoteAddr", value="127.0.0.1:35788")
Annotation(key="Request.ContentLength", value="0")
Annotation(key="Response.Headers.Span-Id", value="8d4bdb285382e850/910810f4ade66b9d/d9564f71aae8aba2")
Annotation(key="Response.ContentLength", value="32")
Annotation(key="Response.StatusCode", value="200")
Annotation(key="Route", value="/endpoint-C")
Annotation(key="User", value="")
Annotation(key="_schema:HTTPServer", value="")
```

### Collection 12
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/910810f4ade66b9d/d9564f71aae8aba2"
    len(anns) = 15

Annotation(key="Request.Proto", value="HTTP/1.1")
Annotation(key="Request.Headers.Span-Id", value="8d4bdb285382e850/910810f4ade66b9d/d9564f71aae8aba2")
Annotation(key="Request.Host", value="localhost:8699")
Annotation(key="Request.RemoteAddr", value="")
Annotation(key="Request.ContentLength", value="0")
Annotation(key="Request.Method", value="GET")
Annotation(key="Request.URI", value="/endpoint-C")
Annotation(key="Response.Headers.Content-Length", value="32")
Annotation(key="Response.Headers.Content-Type", value="text/plain; charset=utf-8")
Annotation(key="Response.Headers.Date", value="Sun, 22 Feb 2015 00:02:29 GMT")
Annotation(key="Response.ContentLength", value="32")
Annotation(key="Response.StatusCode", value="200")
Annotation(key="ClientSend", value="2015-02-21T17:02:29.068068228-07:00")
Annotation(key="ClientRecv", value="2015-02-21T17:02:29.370099164-07:00")
Annotation(key="_schema:HTTPClient", value="")
```

### Collection 13
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/d9564f71aae8aba2"
    len(anns) = 2

Annotation(key="Name", value="/api-calls")
Annotation(key="_schema:name", value="")
```

### Collection 14
```
RemoteCollector.Collect called
    span = "8d4bdb285382e850/d9564f71aae8aba2"
    len(anns) = 21

Annotation(key="Request.ContentLength", value="0")
Annotation(key="Request.Method", value="GET")
Annotation(key="Request.URI", value="/api-calls")
Annotation(key="Request.Proto", value="HTTP/1.1")
Annotation(key="Request.Headers.Cookie", value="_ga=GA1.1.1121926742.1421522090")
Annotation(key="Request.Headers.Connection", value="keep-alive")
Annotation(key="Request.Headers.Cache-Control", value="max-age=0")
Annotation(key="Request.Headers.Accept", value="text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
Annotation(key="Request.Headers.User-Agent", value="Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36")
Annotation(key="Request.Headers.Accept-Encoding", value="gzip, deflate, sdch")
Annotation(key="Request.Headers.Accept-Language", value="en-US,en;q=0.8")
Annotation(key="Request.Host", value="localhost:8699")
Annotation(key="Request.RemoteAddr", value="127.0.0.1:35783")
Annotation(key="Response.Headers.Span-Id", value="8d4bdb285382e850/d9564f71aae8aba2")
Annotation(key="Response.ContentLength", value="226")
Annotation(key="Response.StatusCode", value="200")
Annotation(key="Route", value="/api-calls")
Annotation(key="User", value="")
Annotation(key="ServerRecv", value="2015-02-21T17:02:28.737546589-07:00")
Annotation(key="ServerSend", value="2015-02-21T17:02:29.37244206-07:00")
Annotation(key="_schema:HTTPServer", value="")
```

### Collection 15
```
RemoteCollector.Collect called
    span = "0b5d081c1b09fac1/08b40a40832a70be"
    len(anns) = 2

Annotation(key="Name", value="/favicon.ico")
Annotation(key="_schema:name", value="")
```

### Collection 16
```
RemoteCollector.Collect called
    span = "0b5d081c1b09fac1/08b40a40832a70be"
    len(anns) = 20

Annotation(key="ServerSend", value="2015-02-21T17:02:29.996661592-07:00")
Annotation(key="Request.Proto", value="HTTP/1.1")
Annotation(key="Request.Headers.Accept-Language", value="en-US,en;q=0.8")
Annotation(key="Request.Headers.Cookie", value="_ga=GA1.1.1121926742.1421522090")
Annotation(key="Request.Headers.Connection", value="keep-alive")
Annotation(key="Request.Headers.Accept", value="*/*")
Annotation(key="Request.Headers.User-Agent", value="Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36")
Annotation(key="Request.Headers.Accept-Encoding", value="gzip, deflate, sdch")
Annotation(key="Request.Host", value="localhost:8699")
Annotation(key="Request.RemoteAddr", value="127.0.0.1:35783")
Annotation(key="Request.ContentLength", value="0")
Annotation(key="Request.Method", value="GET")
Annotation(key="Request.URI", value="/favicon.ico")
Annotation(key="Response.ContentLength", value="132")
Annotation(key="Response.StatusCode", value="200")
Annotation(key="Response.Headers.Span-Id", value="0b5d081c1b09fac1/08b40a40832a70be")
Annotation(key="Route", value="/favicon.ico")
Annotation(key="User", value="")
Annotation(key="ServerRecv", value="2015-02-21T17:02:29.996609002-07:00")
Annotation(key="_schema:HTTPServer", value="")
```

