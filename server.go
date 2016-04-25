package stacked

import (
	"bufio"
	"io"
	"log"
	"net"
	"time"
)

// Server serves one or more Detectors.  The first one whose Test function
// returns true wins.
type Server []Detector

// ListenAndServe creates a server for the passed detectors, and has it listend
// and serve.
func ListenAndServe(hostPort string, detectors ...Detector) error {
	return NewServer(detectors...).ListenAndServe(hostPort)
}

// NewServer creates a new Server from a variadic list of Detectors.
func NewServer(detectors ...Detector) Server {
	return Server(detectors)
}

// ListenAndServe opens a listening TCP socket, and calls Serve on it.
func (srv Server) ListenAndServe(hostPort string) error {
	ln, err := net.Listen("tcp", hostPort)
	if err != nil {
		return err
	}
	return srv.Serve(ln)
}

// Serve runs a handling loop on a listening socket.
func (srv Server) Serve(ln net.Listener) error {
	// TODO: afford start-able handlers?  Currently the requirement is that any
	// such need is met lazily/on-demand as connBufShim does.
	defer srv.closeDetectors()

	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Printf("stacked: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0

		go srv.handleConnection(conn)
	}
}

func (srv Server) closeDetectors() {
	for _, det := range srv {
		if closer, ok := det.Handler.(io.Closer); ok {
			closer.Close() // TODO: do we care about err?
		}
	}
}

func (srv Server) handleConnection(conn net.Conn) {
	// TODO: suspect could do better in slow case where we don't have any
	// initial bytes yet... bufr doesn't seem to have a mechanism to wait for X
	// bytes to be available, that then lets us give them all back
	size := 512
	for _, det := range srv {
		if det.Needed > size {
			size = det.Needed
		}
	}
	bufr := bufio.NewReaderSize(conn, size)
	i := 0
	for k := 0; k < 10; k++ {
		for ; i < len(srv); i++ {
			det := srv[i]
			if b, _ := bufr.Peek(det.Needed); len(b) < det.Needed {
				break
			} else if det.Test(b) {
				det.Handler.ServeConnection(conn, bufr)
				return
			}
		}
	}
	log.Printf("stacked: no detector wanted the connection")
	conn.Close()
}
