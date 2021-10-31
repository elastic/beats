package osquery

import (
	"context"
	"time"

	"github.com/osquery/osquery-go/gen/osquery"
	"github.com/osquery/osquery-go/transport"
	"github.com/pkg/errors"

	"github.com/apache/thrift/lib/go/thrift"
)

type ExtensionManager interface {
	Close()
	Ping() (*osquery.ExtensionStatus, error)
	Call(registry, item string, req osquery.ExtensionPluginRequest) (*osquery.ExtensionResponse, error)
	Extensions() (osquery.InternalExtensionList, error)
	RegisterExtension(info *osquery.InternalExtensionInfo, registry osquery.ExtensionRegistry) (*osquery.ExtensionStatus, error)
	DeregisterExtension(uuid osquery.ExtensionRouteUUID) (*osquery.ExtensionStatus, error)
	Options() (osquery.InternalOptionList, error)
	Query(sql string) (*osquery.ExtensionResponse, error)
	GetQueryColumns(sql string) (*osquery.ExtensionResponse, error)
}

// ExtensionManagerClient is a wrapper for the osquery Thrift extensions API.
type ExtensionManagerClient struct {
	Client    osquery.ExtensionManager
	transport thrift.TTransport
}

// NewClient creates a new client communicating to osquery over the socket at
// the provided path. If resolving the address or connecting to the socket
// fails, this function will error.
func NewClient(path string, timeout time.Duration) (*ExtensionManagerClient, error) {
	trans, err := transport.Open(path, timeout)
	if err != nil {
		return nil, err
	}

	client := osquery.NewExtensionManagerClientFactory(
		trans,
		thrift.NewTBinaryProtocolFactoryDefault(),
	)

	return &ExtensionManagerClient{client, trans}, nil
}

// Close should be called to close the transport when use of the client is
// completed.
func (c *ExtensionManagerClient) Close() {
	if c.transport != nil && c.transport.IsOpen() {
		c.transport.Close()
	}
}

// Ping requests metadata from the extension manager.
func (c *ExtensionManagerClient) Ping() (*osquery.ExtensionStatus, error) {
	return c.Client.Ping(context.Background())
}

// Call requests a call to an extension (or core) registry plugin.
func (c *ExtensionManagerClient) Call(registry, item string, request osquery.ExtensionPluginRequest) (*osquery.ExtensionResponse, error) {
	return c.Client.Call(context.Background(), registry, item, request)
}

// Extensions requests the list of active registered extensions.
func (c *ExtensionManagerClient) Extensions() (osquery.InternalExtensionList, error) {
	return c.Client.Extensions(context.Background())
}

// RegisterExtension registers the extension plugins with the osquery process.
func (c *ExtensionManagerClient) RegisterExtension(info *osquery.InternalExtensionInfo, registry osquery.ExtensionRegistry) (*osquery.ExtensionStatus, error) {
	return c.Client.RegisterExtension(context.Background(), info, registry)
}

// DeregisterExtension de-registers the extension plugins with the osquery process.
func (c *ExtensionManagerClient) DeregisterExtension(uuid osquery.ExtensionRouteUUID) (*osquery.ExtensionStatus, error) {
	return c.Client.DeregisterExtension(context.Background(), uuid)
}

// Options requests the list of bootstrap or configuration options.
func (c *ExtensionManagerClient) Options() (osquery.InternalOptionList, error) {
	return c.Client.Options(context.Background())
}

// Query requests a query to be run and returns the extension response.
// Consider using the QueryRow or QueryRows helpers for a more friendly
// interface.
func (c *ExtensionManagerClient) Query(sql string) (*osquery.ExtensionResponse, error) {
	return c.Client.Query(context.Background(), sql)
}

// QueryRows is a helper that executes the requested query and returns the
// results. It handles checking both the transport level errors and the osquery
// internal errors by returning a normal Go error type.
func (c *ExtensionManagerClient) QueryRows(sql string) ([]map[string]string, error) {
	res, err := c.Query(sql)
	if err != nil {
		return nil, errors.Wrap(err, "transport error in query")
	}
	if res.Status == nil {
		return nil, errors.New("query returned nil status")
	}
	if res.Status.Code != 0 {
		return nil, errors.Errorf("query returned error: %s", res.Status.Message)
	}
	return res.Response, nil

}

// QueryRow behaves similarly to QueryRows, but it returns an error if the
// query does not return exactly one row.
func (c *ExtensionManagerClient) QueryRow(sql string) (map[string]string, error) {
	res, err := c.QueryRows(sql)
	if err != nil {
		return nil, err
	}
	if len(res) != 1 {
		return nil, errors.Errorf("expected 1 row, got %d", len(res))
	}
	return res[0], nil
}

// GetQueryColumns requests the columns returned by the parsed query.
func (c *ExtensionManagerClient) GetQueryColumns(sql string) (*osquery.ExtensionResponse, error) {
	return c.Client.GetQueryColumns(context.Background(), sql)
}
