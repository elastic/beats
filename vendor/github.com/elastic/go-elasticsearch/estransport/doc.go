/*
Package estransport provides the transport layer for the Elasticsearch client.

It is automatically included in the client provided by the github.com/elastic/go-elasticsearch package
and is not intended for direct use: to configure the client, use the elasticsearch.Config struct.

At the moment, the implementation is rather minimal: the client takes a slice of url.URL pointers,
and round-robins across them when performing the request.

The default HTTP transport of the client is http.Transport.

*/
package estransport
