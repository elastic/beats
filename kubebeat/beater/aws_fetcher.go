package beater

// Fetcher represents a data fetcher.
type AwsFetcher interface {
	Fetch() ([]interface{}, error)
	Stop()
}

