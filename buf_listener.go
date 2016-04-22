package stacked

import (
	"errors"
	"net"
	"sync/atomic"
)

var errBufListenerClosed = errors.New("bufListener closed")

// bufListener implements net.Listener around a chan *bufConn.
type bufListener struct {
	addr   net.Addr
	conns  chan *bufConn
	closed uint64
}

// Accept waits for and returns the next connection to the listener.
func (bl *bufListener) Accept() (net.Conn, error) {
	if bl.closed != 0 {
		return nil, errBufListenerClosed
	}
	if conn := <-bl.conns; conn != nil {
		return conn, nil
	}
	return nil, errBufListenerClosed
}

// Close closes the listener.
func (bl *bufListener) Close() error {
	if !atomic.CompareAndSwapUint64(&bl.closed, 0, 1) {
		return nil
	}
	close(bl.conns)
	return nil
}

// Addr returns the listener's network address.
func (bl *bufListener) Addr() net.Addr {
	return bl.addr
}
