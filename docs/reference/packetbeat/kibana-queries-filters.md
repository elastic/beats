---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/kibana-queries-filters.html
---

# Kibana queries and filters [kibana-queries-filters]

This topic provides a short introduction to some useful queries for searching Packetbeat data. For a full description of the query syntax, see [Searching Your Data](elasticsearch://reference/query-languages/kql.md) in the *Kibana User Guide*.

In Kibana, you can filter transactions either by entering a search query or by clicking on elements within a visualization.


## Create queries [_create_queries]

The search field on the **Discover** page provides a way to query a specific subset of transactions from the selected time frame. It allows boolean operators, wildcards, and field filtering. For example, if you want to find the HTTP redirects, you can search for `http.response.status_code: 302`.

:::{image} images/kibana-query-filtering.png
:alt: Kibana query
:class: screenshot
:::


### String queries [_string_queries]

A query may consist of one or more words or a phrase. A phrase is a group of words surrounded by double quotation marks, such as `"test search"`.

To search for all HTTP requests initiated by Mozilla Web browser version 5.0:

```yaml
"Mozilla/5.0"
```

To search for all the transactions that contain the following message:

```yaml
"Cannot change the info of a user"
```

::::{note}
To search for an exact string, you need to wrap the string in double quotation marks. Without quotation marks, the search in the example would match any documents containing one of the following words: "Cannot" OR "change" OR "the" OR "info" OR "a" OR "user".
::::


To search for all transactions with the "chunked" encoding:

```yaml
"Transfer-Encoding: chunked"
```


### Field-based queries [_field_based_queries]

Kibana allows you to search specific fields.

To view HTTP transactions only:

```yaml
type: http
```

To view failed transactions only:

```yaml
status: Error
```

To view INSERT queries only:

```yaml
method: INSERT
```


### Regexp queries [_regexp_queries]

Kibana supports regular expression for filters and expressions. For example, to search for all HTTP responses with JSON as the returned value type:

```yaml
http.response_headers.content_type: *json
```

See [Elasticsearch regexp query](elasticsearch://reference/query-languages/query-dsl/query-dsl-regexp-query.md) for more details about the syntax.


### Range queries [_range_queries]

Range queries allow a field to have values between the lower and upper bounds. The interval can include or exclude the bounds depending on the type of brackets that you use.

To search for slow transactions with a response time greater than or equal to 10ms:

```yaml
event.duration: [10000000 TO *]
```

To search for slow transactions with a response time greater than 10ms:

```yaml
responsetime: {10000000 TO *}
```


### Boolean queries [_boolean_queries]

Boolean operators (AND, OR, NOT) allow combining multiple sub-queries through logic operators.

::::{note}
Operators such as AND, OR, and NOT must be capitalized.
::::


To search for all transactions except MySQL transactions:

```yaml
NOT type: mysql
```

To search for all MySQL INSERT queries with errors:

```yaml
type: mysql AND method: INSERT AND status: Error
```

Kibana Query Language (KQL) also supports parentheses to group sub-queries.

To search for either INSERT or UPDATE queries with a response time greater than or equal to 30ms:

```yaml
(method: INSERT OR method: UPDATE) AND event.duration >= 30000000
```


## Create filters [_create_filters]

In Kibana, you can also filter transactions by clicking on elements within a visualization. For example, to filter for all the HTTP redirects that are coming from a specific IP and port, click the **Filter for value** ![filterforval icon](images/filterforval_icon.png "") icon next to the `client.ip` and `client.port` fields in the transaction detail table. To exclude the HTTP redirects coming from the IP and port, click the **Filter out value** ![filteroutval icon](images/filteroutval_icon.png "") icon instead.

:::{image} images/filter_from_context.png
:alt: Filter from context
:class: screenshot
:::

The selected filters appear under the search box.

:::{image} images/kibana-filters.png
:alt: Kibana filters
:class: screenshot
:::

