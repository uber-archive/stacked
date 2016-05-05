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
	"errors"
	"net"
	"sync/atomic"
)

var errBufListenerClosed = errors.New("bufListener closed")

// bufListener implements net.Listener around a chan *bufConn.
type bufListener struct {
	addr   net.Addr
	conns  chan net.Conn
	closed uint64
}

func newBufListener(addr net.Addr) *bufListener {
	return &bufListener{
		addr:  addr,
		conns: make(chan net.Conn),
	}
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
