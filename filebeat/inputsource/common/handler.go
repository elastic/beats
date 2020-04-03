package common

import (
	"net"
)

// HandlerFactory returns a ConnectionHandler func
type HandlerFactory func(config ListenerConfig) ConnectionHandler

// ConnectionHandler interface provides mechanisms for handling of incoming connections
type ConnectionHandler interface {
	Handle(CloseRef, net.Conn) error
}
