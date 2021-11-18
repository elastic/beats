package beater

// Fetcher represents a data fetcher.
type Fetcher interface {
	Fetch() ([]interface{}, error)
	Stop()
}
