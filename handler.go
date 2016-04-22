package stacked

import (
	"bufio"
	"net"
)

// Handler is a higher level interface implemented by handlers that can
// re-use the buffered reader.
type Handler interface {
	ServeConnection(conn net.Conn, bufr *bufio.Reader)
	// TODO: maybe shift to a struct{net.Conn, bufio.Reader}
}

// HandlerFunc is a convenience type for implementing simple ConnHandlers.
type HandlerFunc func(conn net.Conn, bufr *bufio.Reader)

// ServeConnection simply calls the function
func (bchf HandlerFunc) ServeConnection(conn net.Conn, bufr *bufio.Reader) {
	bchf(conn, bufr)
}
