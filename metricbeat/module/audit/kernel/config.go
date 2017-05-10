package kernel

// Config defines the kernel metricset's possible configuration options.
type Config struct {
	ResolveIDs   bool   `config:"kernel.resolve_ids"`         // Resolve UID/GIDs to names.
	BacklogLimit uint32 `config:"kernel.backlog_limit"`       // Max number of message to buffer in the kernel.
	RateLimit    uint32 `config:"kernel.rate_limit"`          // Rate limit in messages/sec of messages from kernel.
	RawMessage   bool   `config:"kernel.include_raw_message"` // Include the list of raw audit messages in the event.
	Warnings     bool   `config:"kernel.include_warnings"`    // Include warnings in the event (for dev/debug purposes only).
}

var defaultConfig = Config{
	ResolveIDs:   true,
	BacklogLimit: 8192,
	RateLimit:    0,
	RawMessage:   false,
	Warnings:     false,
}
