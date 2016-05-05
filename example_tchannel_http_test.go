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

package stacked_test

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"log"
	"net/http"

	"github.com/uber/tchannel-go"

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
		return false
	}
	if frameSize < 20 {
		// not enough for a zero-header frame
		return false
	}
	// TODO: surely we can set some upper bound << 64Ki

	// type:1
	if err := binary.Read(buf, binary.BigEndian, &frameType); err != nil {
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
		return false
	}
	// TODO: is this too restrictive? surely we can at least say something like <256...
	if tchanVersion != 2 {
		return false
	}

	// nh:2
	if err := binary.Read(buf, binary.BigEndian, &numHeaders); err != nil {
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

// openssl genrsa -out server.key 2048
const serverKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpgIBAAKCAQEA3bM2koA8pCZReGNQ97Z7qCgYhF/XZkA9ZbfhTqXva/4R82LR
K8/I/jF0/S8pD/8eUd6/mms6ydTu5VbLLcZCR3n+gNpBxTCYOAUUnEtVsvFJvlDK
A1byaj5POX5cKFRjR3BdCiGkD6YNF5hrLF1sHIp02AOzolII/EAx+cN8gkyMW8zc
VYsimZ4uKKgxfL6nUDUlwYt+PIJ2wA0zMR2LrlIsRnUpIjGeOkg7KVyLkSNmIZbN
aJjJK5Nj1oZVFBrM14VbBrUCkcX9iOqnDEq/39p5Pk/A9QFbYhMTYjz6sp4PXD2w
SnGEYX8EzMB98+1LRbpiQekZR4bqju3jaKpW0QIDAQABAoIBAQCGnpLtpIauGkJw
OsZolFtEAYzZnKTcBvgBMwXRzvqx9aYKxx9CXjqq93cVYjSp7P0JM5ve9WvOMMkb
Y3eehPusEUzUCzPSvC5CHfuk6C3SqadgtAfmvT4X+1v6CluFdbCPKZClXUYU5nye
rkOtvdCvB/fpT14dm3ivS3/NLMIHD7hi9UIv18OTtPIMxhNBuZfKM+Vm4SvJNtkY
/EBfQCM+1aYb61gs+DEWE2Svut4D+gkff1XgMP1w4/Cv+J04VYf3pVQWS3P+MotB
d+ab247K/9w3JknnewWBCEYdWeyx1LneMhpI4Q+magDTMzNnfWLdRDma9zYB7DO1
YDKvAiitAoGBAP4TAX84OERpwDWfn3lpm9fli0lLXbGxcCMqBTrFtRyXSo6XJycC
fFkEsZS1moxth/QE1r1xbIliGrSOPWCHZpe3HlQBixLgVh4wJ+RhbHZheXat1hYk
fmAZQjXzQL6QPixHxf/3zbjGzlZVThwQRDrsEN1mJW4WVeA1iYYcT+HvAoGBAN9h
Y9FWnbTVNDGIf9NEqG66/nhabGMzbmFaixDIz7ZKTRrxo3niaZLAKHEuEoNoiuZG
gpfLVUFZ2bw07sOJg8jdcJUEfCDlC5YCw4t0cBlYgqI+UrCFo4hSDHI5XcInZWBU
kn+Pn0UBErZEuqckZzUQwBYvNsfK0oeVH22pkxM/AoGBAM5kthSYoOzCU0e8UZoZ
dmXdrFZwCL6ue3+1ROZHcSa2p/RJSZ7g4A6YR4GcPN3SpFxQCfl+yEKaFUOTQLzH
gUnBkbuAPW+qGYsQZ3eYxLkt2bPU51K5doeuPSECaBflqPvjmi6jKNTvevKa/YbC
mAqddd3EeqeBMWWfWAY/vYy1AoGBAKTogwZCSX78fuGqgaN4ZlgI2GAFcUry5yQb
8dpcRWuwAqhHh4Ytrf7WuYSEnMpCXXiOyU5CoBf0uxeEhFf6pz2crMZ2XyTxstH2
DGJhfXhYrWgVVnpWzlmPKP0SeLMi3mZ1SQm+/7eziRriNmG6MC8uxIAcLvbkNvQ9
FMyiiZ+FAoGBANMKK8ZXs+8j2f5vhojjvdUF4oPH9/ixIlnRwxy3EEj06MGjeWEt
pebz3YFd3OSmE2gxhavQWCkvwpOIIfEjjUc41idshfbIJyMx47YlGbeuhJl/3vXZ
BqW0u6WlCI24+oF08MF4WYysdVrL4EkLNxVfR5uYyhMZAxoTFF9JBC80
-----END RSA PRIVATE KEY-----
`

// openssl req -new -x509 -key server.key -out server.pem -days 3650
const serverCert = `
-----BEGIN CERTIFICATE-----
MIIEUTCCAzmgAwIBAgIJAKWzCk2UhplZMA0GCSqGSIb3DQEBBQUAMHgxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIEwpTb21lLVN0YXRlMSEwHwYDVQQKExhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQxEjAQBgNVBAMTCWxvY2FsaG9zdDEdMBsGCSqGSIb3DQEJ
ARYOcm9vdEBsb2NhbGhvc3QwHhcNMTYwNDI2MDQzODIzWhcNMjYwNDI0MDQzODIz
WjB4MQswCQYDVQQGEwJBVTETMBEGA1UECBMKU29tZS1TdGF0ZTEhMB8GA1UEChMY
SW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMRIwEAYDVQQDEwlsb2NhbGhvc3QxHTAb
BgkqhkiG9w0BCQEWDnJvb3RAbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEA3bM2koA8pCZReGNQ97Z7qCgYhF/XZkA9ZbfhTqXva/4R82LR
K8/I/jF0/S8pD/8eUd6/mms6ydTu5VbLLcZCR3n+gNpBxTCYOAUUnEtVsvFJvlDK
A1byaj5POX5cKFRjR3BdCiGkD6YNF5hrLF1sHIp02AOzolII/EAx+cN8gkyMW8zc
VYsimZ4uKKgxfL6nUDUlwYt+PIJ2wA0zMR2LrlIsRnUpIjGeOkg7KVyLkSNmIZbN
aJjJK5Nj1oZVFBrM14VbBrUCkcX9iOqnDEq/39p5Pk/A9QFbYhMTYjz6sp4PXD2w
SnGEYX8EzMB98+1LRbpiQekZR4bqju3jaKpW0QIDAQABo4HdMIHaMB0GA1UdDgQW
BBTc7YWhT0GImjx2uyp2NWs7P2g4WjCBqgYDVR0jBIGiMIGfgBTc7YWhT0GImjx2
uyp2NWs7P2g4WqF8pHoweDELMAkGA1UEBhMCQVUxEzARBgNVBAgTClNvbWUtU3Rh
dGUxITAfBgNVBAoTGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDESMBAGA1UEAxMJ
bG9jYWxob3N0MR0wGwYJKoZIhvcNAQkBFg5yb290QGxvY2FsaG9zdIIJAKWzCk2U
hplZMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEFBQADggEBAMpZnPniz8p/yVVq
HCvHWENWzvjQDiL9XksyfqyaCdXa+7nY905m2g4vAJk6Y8PjppmOjQOoSa3q8ZWg
kGg+1fMsbCnngUp0un7Bfbz58+ctEvrvhIOptZqTu0sk5kMWs10ABXQy5A8arCI8
egI13OiFrUblOdt22KwcMLSqQgM/vwAD/GsUTGbpBUMsngLCwXSvGzJaRhDZWB/o
xHVNJi29343lXgNYQT64gnbauNDhrKUJgfG+bQtuAMaL8m0/AFsjwRQqqNRD5eJ0
wF9Vlw5GV8NyPh/54e82pwLAovwo3959TymBG5VH0uPNeR15dxhAzgrG8RikZtqF
JaN36yE=
-----END CERTIFICATE-----
`

func main() {
	cer, err := tls.X509KeyPair([]byte(serverCert), []byte(serverKey))
	if err != nil {
		log.Fatal(err)
	}

	ch, err := tchannel.NewChannel("foo", nil)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Handler: http.DefaultServeMux,
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cer},
	}

	log.Fatal(stacked.ListenAndServe(":4040",
		// will serve tchannel protocol first if we get what looks like a valid init frame
		stacked.Detector{
			Needed:  20,
			Test:    isTChannelInitFrame,
			Handler: stacked.ListenServerHandler(ch),
		},

		// detect TLS client handshake, serve http default mux
		stacked.TLSServer(config, srv),

		// otherwise will serve default HTTP
		stacked.DefaultHTTPHandler(http.DefaultServeMux),
	))
}
