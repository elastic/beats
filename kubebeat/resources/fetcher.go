package resources

// Fetcher represents a data fetcher.
type Fetcher interface {
	Fetch() ([]FetcherResult, error)
	Stop()
}

type FetcherResult struct {
	Type     string      `json:"type"`
	Resource interface{} `json:"resource"`
}
