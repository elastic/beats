package memqueue

type logger interface {
	Debug(...interface{})
	Debugf(string, ...interface{})
}
