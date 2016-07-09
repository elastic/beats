package beat

// FlagsHandler is an interface that can optionally be implemented by a Beat
// if it needs to process command line flags on startup. If implemented, the
// HandleFlags method will be invoked after parsing the command line flags
// and before any of the Beater interface methods are invoked. There will be
// no callback when '-help' or '-version' are specified.
type FlagsHandler interface {
	HandleFlags(*Beat) error // Handle any custom command line arguments.
}

type FlagsHandlerCallback func(*Beat) error

var handlers []FlagsHandler

func AddFlagsHandler(h FlagsHandler) {
	handlers = append(handlers, h)
}

func AddFlagsCallback(cb func(*Beat) error) {
	AddFlagsHandler(FlagsHandlerCallback(cb))
}

func handleFlags(b *Beat) error {
	for _, h := range handlers {
		err := h.HandleFlags(b)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cb FlagsHandlerCallback) HandleFlags(b *Beat) error {
	return cb(b)
}
