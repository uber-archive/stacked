package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/uber-common/stacked"
)

// Implements a dual-protocol TODO notes server.  The server:
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
	log.Printf("NOTE: %#v", note)
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
			if len(b) == 1 && b[0] == '.' {
				io.WriteString(conn, "# Notes:\n")
				for _, note := range notes {
					io.WriteString(conn, fmt.Sprintf("- %s\n", note))
				}
				io.WriteString(conn, "\n")
				continue
			}

			log.Printf("wat %#v\n", string(b))
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

func main() {
	log.Fatal(stacked.NewServer(
		stacked.Detector{
			Needed:  2,
			Test:    looksLikeANote,
			Handler: stacked.HandlerFunc(handleMarked),
		},
		stacked.DefaultHTTPHandler(http.HandlerFunc(handleHTTP)),
	).ListenAndServe(":4040"))
}
