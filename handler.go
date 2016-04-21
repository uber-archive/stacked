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

// ConnHandlerFunc is a convenience type for implementing simple ConnHandlers.
type ConnHandlerFunc func(conn net.Conn, bufr *bufio.Reader)

// ServeConnection simply calls the function
func (bchf ConnHandlerFunc) ServeConnection(conn net.Conn, bufr *bufio.Reader) {
	bchf(conn, bufr)
}
