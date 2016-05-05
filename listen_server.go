// Copyright (c) 2016 Uber Technologies, Inc
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
	conn = &bufConn{conn, bufr}
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
