package stacked

import (
	"bufio"
	"net"
)

// ListenServer is the minimal downstream server interface, e.g. implemented by
// http.Server.
type ListenServer interface {
	Serve(ln net.Listener) error
}

// ListenServerHandler creates a compatibility Handler for
func ListenServerHandler(srv ListenServer) Handler {
	return &connBufShim{Server: srv}
}

// connBufShim implements Handler interface around a ListenServer
type connBufShim struct {
	Server    ListenServer
	listeners map[net.Addr]*bufListener
}

// ServeConnection simply puts a new bufConn onto bufConns for distribution by
// bufLn.Accept.
func (cbs *connBufShim) ServeConnection(conn net.Conn, bufr *bufio.Reader) {
	cbs.lnFor(conn).conns <- &bufConn{conn, bufr}
}

// lnFor gets or creates the bufListener for connection (one-per
// conn.LocalAddr()).
func (cbs *connBufShim) lnFor(conn net.Conn) *bufListener {
	addr := conn.LocalAddr()
	if cbs.listeners == nil {
		cbs.listeners = make(map[net.Addr]*bufListener, 1)
	}
	if ln := cbs.listeners[addr]; ln != nil {
		return ln
	}
	ln := newBufListener(addr)
	cbs.listeners[addr] = ln
	go func() {
		if err := cbs.Server.Serve(ln); err != nil {
			delete(cbs.listeners, addr)
		}
	}()
	return ln
}

// Close closes any bufListeners
func (cbs *connBufShim) Close() error {
	if cbs.listeners == nil {
		for _, ln := range cbs.listeners {
			ln.Close() // TODO: care about error? use a MultiError?
		}
		cbs.listeners = nil
	}
	return nil
}
