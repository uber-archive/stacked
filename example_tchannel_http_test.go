package stacked_test

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"log"
	"net"

	"github.com/uber-common/stacked"
)

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
	log.Fatal(stacked.NewServer(
		stacked.Detector{
			Needed: 20,
			Test:   isTChannelInitFrame,
			// TODO: currently we just close the connection, since tchannel-go
			// integration is unclear (it doesn't seem to export a ListenServer
			// implementation).
			Handler: stacked.HandlerFunc(func(conn net.Conn, bufr *bufio.Reader) {
				log.Printf("no tchannel for you!")
				conn.Close()
			}),
		}, // will serve marked protocol first if initial bytes "look like a note"
		stacked.DefaultHTTPHandler(nil), // otherwise will serve default HTTP
	).ListenAndServe(":4040"))
}
