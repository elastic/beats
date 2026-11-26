{applies_to}`stack: ga 9.7.0` The request tracer masks the values of credentials that it writes, but it does not remove all sensitive data from the trace. Masked values are replaced with `REDACTED`.

The tracer masks:

- The values of the `Authorization`, `Proxy-Authorization`, `Cookie`, `Set-Cookie`, `X-Api-Key` and `X-ApiKeys` headers, and of any other header whose name contains a word that suggests a credential, such as `token`, `key` or `secret`. The scheme of an `Authorization` or `Proxy-Authorization` value, for example `Bearer` or `Basic`, stays visible.
- The values of the `access_token`, `api_key`, `api_token`, `apikey`, `assertion`, `auth_token`, `client_assertion`, `client_secret`, `id_token`, `key`, `password`, `refresh_token`, `secret`, `sig`, `signature`, `token`, `X-Amz-Security-Token` and `X-Amz-Signature` URL query parameters, and the user name and password in a URL.
- The values of the `access_token`, `api_key`, `apikey`, `assertion`, `client_assertion`, `client_secret`, `id_token`, `password`, `refresh_token`, `secret` and `token` fields, at any depth, in form and JSON request and response bodies, including newline-delimited JSON. Names are matched without regard to case. If one of these fields holds an object or an array, every string within it is masked.

When the tracer masks a value in a form or JSON body, it writes the body again: JSON object keys are sorted, whitespace is removed, and form parameters are sorted. To do this it decodes the whole body, which adds CPU and memory use for large responses while the tracer is on.

The tracer does not mask:

- Credentials in body fields, query parameters or headers that have other names.
- Bodies that are neither form nor JSON, such as XML or plain text.

If a form or JSON body contains one of the field names above but cannot be parsed, the tracer logs a fixed message in place of the body.

Trace files are plain files in the logs directory. When the input runs under {{agent}}, diagnostics bundles include them as they were written, so the masking described here is the only redaction applied to them. Review a trace file before you share it.
