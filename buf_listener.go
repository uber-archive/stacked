package stacked

import (
	"errors"
	"net"
)

var errBufListenerClosed = errors.New("bufListener closed")

// bufListener implements net.Listener around a chan *bufConn.
type bufListener struct {
	addr   net.Addr
	inc    chan *bufConn
	closed bool
}

// Accept waits for and returns the next connection to the listener.
func (bl *bufListener) Accept() (net.Conn, error) {
	if bl.closed {
		return nil, errBufListenerClosed
	}
	if conn := <-bl.inc; conn != nil {
		return conn, nil
	}
	bl.closed = true
	return nil, errBufListenerClosed
}

// Close closes the listener.
func (bl *bufListener) Close() error {
	bl.closed = true
	// TODO: interrupt / unblock any blocked accept operations
	return nil
}

// Addr returns the listener's network address.
func (bl *bufListener) Addr() net.Addr {
	return bl.addr
}
