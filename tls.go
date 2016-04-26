package stacked

import (
	"bufio"
	"crypto/tls"
	"net"
)

// minimum length with zero sessionId, cipherSuite, compressionMethod, and
// extension is 49
const minBytes = 49

// isTLSClientHello checks whether the passed bytes can be (at least a prefix
// of) a TLS client hello message
func isTLSClientHello(data []byte) bool {
	// TODO: audit against https://tools.ietf.org/html/rfc5246#section-7.3, for
	// now I've used tls.clientHelloMsg.unmarshal as my "documentation" ;-)

	if len(data) < minBytes {
		return false
	}

	// recordType:1
	if data[0] != 0x16 { // recordTypeHandshake
		return false
	}
	data = data[1:]

	// vers:2
	vers := int(data[0])<<8 | int(data[1])
	if vers < tls.VersionSSL30 || vers > tls.VersionTLS12 {
		return false
	}
	data = data[2:]

	// len:2
	// we may not have all of the handshake data, so we can't always strictly
	// the length, but we'll track remaining declared bytes and progressively
	// make assertions about it
	remain := int(data[0])<<8 | int(data[1])
	if remain < 44 {
		return false
	}
	data = data[2:]

	// messageType:1
	if data[0] != 0x01 {
		return false
		// type 1 is client hello
	}
	remain--

	// len:3
	length := int(data[1])<<16 | int(data[2])<<8 | int(data[3])
	remain -= 3

	// minimum hello message size is 40 (when all variable fields are zero-width)
	if remain < length {
		return false
	}

	// vers:2
	vers = int(data[4])<<8 | int(data[5])
	if vers < tls.VersionSSL30 || vers > tls.VersionTLS12 {
		return false
	}
	remain -= 2

	// random:32
	// skip
	data = data[38:]
	remain -= 32

	// sessionId~1
	sessionIDLen := int(data[0])
	remain--
	if remain < sessionIDLen { // can't contain the claimed session id
		return false
	}
	remain -= sessionIDLen
	data = data[1+sessionIDLen:]

	// (cipherSuite:2)~2
	if len(data) < 2 {
		// TBD: didn't have enough to do full verification, be optimistic
		return true
	}
	cipherSuiteLen := int(data[0])<<8 | int(data[1])
	remain -= 2
	data = data[2:]
	if cipherSuiteLen%2 == 1 { // must have even number since uint16 array
		return false
	}
	if remain < cipherSuiteLen { // can't contain the claimed cipherSuite array
		return false
	}
	if len(data) < cipherSuiteLen {
		// TBD: didn't have enough to do full verification, be optimistic
		return true
	}
	remain -= cipherSuiteLen
	data = data[cipherSuiteLen:]

	// (compressionMethod:1)~1
	if len(data) < 1 {
		// TBD: didn't have enough to do full verification, be optimistic
		return true
	}
	remain--
	data = data[1:]

	compressionMethodsLen := int(data[0])
	if remain < compressionMethodsLen { // can't contain the claimed compressionMethods array
		return false
	}
	if len(data) < compressionMethodsLen {
		// TBD: didn't have enough to do full verification, be optimistic
		return true
	}
	remain -= compressionMethodsLen
	data = data[compressionMethodsLen:]

	// (extension:2 data~2)~2
	if len(data) < 2 {
		// TBD: didn't have enough to do full verification, be optimistic
		return true
	}
	extensionsLength := int(data[0])<<8 | int(data[1])
	data = data[2:]
	remain -= 2
	if remain < extensionsLength { // can't contain the claimed extensions array
		return false
	}

	for len(data) != 0 {
		if len(data) < 4 {
			// TBD: didn't have enough to do full verification, be optimistic
			return true
		} else if remain < 4 { // declared length can't contain another extension
			return false
		}

		// TODO: could check by extension type:
		// extension := int(data[0])<<8 | int(data[1])

		length := int(data[2])<<8 | int(data[3])
		data = data[4:]
		remain -= 4
		if remain < length { // declared length can't contain this extension data
			return false
		}
		if len(data) < length {
			// TBD: didn't have enough to do full verification, be optimistic
			return true
		}
		data = data[length:]
		remain -= length
	}

	if remain > 0 { // leftover declared bytes
		return false
	}

	return true
}

type tlsShim struct {
	connBufShim
	config *tls.Config
}

func newTLSShim(config *tls.Config, srv ListenServer) *tlsShim {
	return &tlsShim{connBufShim{Server: srv}, config}
}

func (ts *tlsShim) ServeConnection(conn net.Conn, bufr *bufio.Reader) {
	conn = &bufConn{conn, bufr}
	conn = tls.Server(conn, ts.config)
	ts.lnFor(conn).conns <- conn
}

// TLSServer returns a detector that detects a client TLS handshake before
// wrapping each connection in tls.Server to pass to the ListenServer.
func TLSServer(config *tls.Config, srv ListenServer) Detector {
	// TODO: isTLSClientHello can really benefit from more bytes
	return Detector{
		Needed:  minBytes,
		Test:    isTLSClientHello,
		Handler: newTLSShim(config, srv),
	}
}
