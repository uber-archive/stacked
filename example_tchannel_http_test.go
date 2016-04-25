package stacked_test

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"log"
	"net"

	"github.com/uber/tchannel-go"

	"github.com/uber-common/stacked"
)

func prefixDetector(p string, handler stacked.Handler) stacked.Detector {
	return stacked.Detector{
		Needed:  len([]byte(p)),
		Test:    func(b []byte) bool { return string(b) == prefix },
		Handler: handler,
	}
}

func isTChannelInitFrame(b []byte) bool {
	buf := bytes.NewBuffer(b)

	var (
		frameSize    uint16
		frameType    uint8
		frameID      uint32
		tchanVersion uint16
		numHeaders   uint16
	)

	// size:2
	if err := binary.Read(buf, binary.BigEndian, &frameSize); err != nil {
		log.Printf("no frameSize: %v", err)
		return false
	}
	if frameSize < 20 {
		// not enough for a zero-header frame
		return false
	}
	// TODO: surely we can set some upper bound << 64Ki

	// type:1
	if err := binary.Read(buf, binary.BigEndian, &frameType); err != nil {
		log.Printf("no frameType: %v", err)
		return false
	}
	if frameType != 0x01 { // init req
		return false
	}

	// reserved:1
	if len(buf.Next(1)) != 1 {
		return false
	}

	// id:4
	if err := binary.Read(buf, binary.BigEndian, &frameID); err != nil {
		log.Printf("no frameID: %v", err)
		return false
	}
	// TODO: should we be able to assert that id is 0 or 1... or at least bound
	// it?  but that would preclude full randomizatino of ids...

	// reserved:8
	if len(buf.Next(8)) != 8 {
		return false
	}

	// version:2
	if err := binary.Read(buf, binary.BigEndian, &tchanVersion); err != nil {
		log.Printf("no tchanVersion: %v", err)
		return false
	}
	// TODO: is this too restrictive? surely we can at least say something like <256...
	if tchanVersion != 2 {
		return false
	}

	// nh:2
	if err := binary.Read(buf, binary.BigEndian, &numHeaders); err != nil {
		log.Printf("no numHeaders: %v", err)
		return false
	}
	if numHeaders == 0 && frameSize != 20 {
		return false
	}

	// (key~2 value~2){nh}

	// TODO: since we express that we only "need" 20, and stacked.Server
	// doesn't volunteer any more than needed, we currently cannot
	// opportunistically verify any headers.  Alternatively, if we had a more
	// complex detector protocol the function could return a "maybe, give me
	// more bytes" response... which would probably be better off just
	// injecting the *bufio.Reader with the contractof "don't advance it"

	return true
}

// Example_tchannelAndHTTP shows how to host both tchannel and http on a single port
func Example_tchannelAndHTTP() {
	ch, err := tchannel.NewChannel("foo", nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(stacked.ListenAndServe(":4040",
		// will serve tchannel protocol first if we get what looks like a valid init frame
		stacked.Detector{
			Needed:  20,
			Test:    isTChannelInitFrame,
			Handler: stacked.ListenServerHandler(ch),
		},

		// detect prior-knowledge HTTP/2 connection... deny for now (this is
		// only for example, and is actually counterproductive since net/http
		// (the next handler below) will handle this fine in 1.6
		prefixDetector(
			"PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n",
			stacked.HandlerFunc(func(conn net.Conn, bufr *bufio.Reader) {
				log.Printf("no HTTP/2 for you!")
				conn.close()
			})),

		// otherwise will serve default HTTP
		stacked.DefaultHTTPHandler(nil),
	))
}
