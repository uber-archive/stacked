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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/uber-common/stacked"
)

var (
	errMalformedNotes = errors.New("malformed notes")
	notes             []string
)

func looksLikeANote(b []byte) bool {
	return len(b) >= 2 && b[0] == '-' && b[1] == ' '
}

func got(b []byte) {
	note := string(b)
	notes = append(notes, note)
}

func handleMarked(conn net.Conn, bufr *bufio.Reader) {
	if err := func() error {
		for {
			b, err := bufr.ReadBytes('\n')
			if err == io.EOF {
				return nil
			} else if err != nil {
				return err
			}
			b = b[:len(b)-1]

			// skip blanks
			if len(b) == 0 {
				continue
			}

			// note submission: "- XXX"
			if looksLikeANote(b) {
				got(b[2:])
				continue
			}

			// dump notes: "."
			if string(b) == "." {
				io.WriteString(conn, "# Notes:\n")
				for _, note := range notes {
					if _, err := io.WriteString(conn, fmt.Sprintf("- %s\n", note)); err != nil {
						return err
					}
				}
				if _, err := io.WriteString(conn, "\n"); err != nil {
					return err
				}
				continue
			}

			// close on "quit"
			if string(b) == "quit" {
				if _, err := io.WriteString(conn, "# goodbye\n"); err != nil {
					return err
				}
				return conn.Close()
			}

			return errMalformedNotes
		}
	}(); err != nil {
		io.WriteString(conn, fmt.Sprintf("# %v\n", err))
		conn.Close()
	}
}

func handleHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.WriteHeader(http.StatusOK)
		for _, note := range notes {
			io.WriteString(w, fmt.Sprintf("- %s\n", note))
		}

	case "POST":
		if err := func() error {
			bufr := bufio.NewReader(r.Body)
			for {
				b, err := bufr.ReadBytes('\n')
				if err == io.EOF {
					return nil
				} else if err != nil {
					return err
				} else if !looksLikeANote(b) {
					return errMalformedNotes
				}
				got(b[2 : len(b)-1])
			}
		}(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, fmt.Sprintf("%s\n%s\n", http.StatusText(http.StatusBadRequest), err))
			return
		}
	}
}

func exampleListenAndServe(hostPort string) {
	markedDetector := stacked.Detector{
		Needed:  2,
		Test:    looksLikeANote,
		Handler: stacked.HandlerFunc(handleMarked),
	}

	httpDetector := stacked.DefaultHTTPHandler(http.HandlerFunc(handleHTTP))

	log.Fatal(stacked.ListenAndServe(hostPort,
		markedDetector, // will serve marked protocol first if initial bytes "look like a note"
		httpDetector,   // otherwise will serve classic HTTP
	))
}

// Example_todoserver implements a dual-protocol TODO notes server.
//
// The server:
// - accepts notes HTTP POSTed to it as markdown-esque lists
// - returns the current note list in response to an HTTP GET
// - supports a raw markdown-esque protoctol, f.e. from netcat
//
// Example:
//   $ curl -X POST localhost:4040 --data-binary @- <<EOF
//   - foo
//   EOF
//
//   $ curl localhost:4040
//   - foo
//
//   $ nc localhost 4040
//   - bar
//   .
//   # Notes:
//   - foo
//   - bar
//
//   $ curl localhost:4040
//   - foo
//   - bar
func Example_todoserver() {
	go exampleListenAndServe("localhost:4040")

	var buf bytes.Buffer

	//// First show that we can use it REST-fully:
	fmt.Println(">>> Initial HTTP GET")
	resp := mustHTTP(http.Get("http://localhost:4040"))
	mustCopyAndClose(os.Stdout, resp.Body)
	fmt.Println()

	fmt.Println(">>> Initial HTTP PUT foo")
	mustWrite(buf.WriteString("- foo\n"))
	resp = mustHTTP(http.Post("http://localhost:4040", "text/plain", &buf))
	mustCopyAndClose(os.Stdout, resp.Body)
	fmt.Println()

	fmt.Println(">>> HTTP GET After POST")
	resp = mustHTTP(http.Get("http://localhost:4040"))
	mustCopyAndClose(os.Stdout, resp.Body)
	fmt.Println()

	//// Now show that we can just use a tcp stream
	fmt.Println(">>> Send second note over raw tcp")
	conn := mustConn(net.Dial("tcp", "localhost:4040"))
	mustWrite(io.WriteString(conn, "- bar\n"))

	fmt.Println(">>> Get notes over raw tcp")
	mustWrite(io.WriteString(conn, ".\n"))

	mustWrite(io.WriteString(conn, "quit\n"))
	mustCopyAndClose(os.Stdout, conn)

	//// Finish with a final HTTP GET to show that it's all the same thing
	fmt.Println(">>> HTTP GET After TCP")
	resp = mustHTTP(http.Get("http://localhost:4040"))
	mustCopyAndClose(os.Stdout, resp.Body)
	fmt.Println()

	fmt.Println(">>> done")

	// Output: >>> Initial HTTP GET
	//
	// >>> Initial HTTP PUT foo
	//
	// >>> HTTP GET After POST
	// - foo
	//
	// >>> Send second note over raw tcp
	// >>> Get notes over raw tcp
	// # Notes:
	// - foo
	// - bar
	//
	// # goodbye
	// >>> HTTP GET After TCP
	// - foo
	// - bar
	//
	// >>> done
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func mustHTTP(resp *http.Response, err error) *http.Response {
	must(err)
	return resp
}

func mustConn(resp net.Conn, err error) net.Conn {
	must(err)
	return resp
}

func mustWrite(_ int, err error) {
	must(err)
}

func mustCopy(dst io.Writer, src io.Reader) {
	_, err := io.Copy(dst, src)
	must(err)
}

func mustCopyAndClose(dst io.Writer, src io.Reader) {
	mustCopy(dst, src)
	if closer, ok := src.(io.Closer); ok {
		must(closer.Close())
	}
}
