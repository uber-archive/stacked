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
	"net/http"
)

// Detector is a Handler along with a detection function and how
// many bytes it needs to decide.
type Detector struct {
	// Needed is how many bytes are needed for the Test function
	Needed int

	// Test will be called with at least Needed bytes.  If this function
	// returns true, then the connection will be given to Handler.
	Test func(b []byte) bool

	Handler Handler
}

// DefaultHTTPHandler creates a FallthroughDetector around an http.Handler.
func DefaultHTTPHandler(hndl http.Handler) Detector {
	if hndl == nil {
		hndl = http.DefaultServeMux
	}
	handler := ListenServerHandler(&http.Server{
		Handler: hndl,
	})
	return FallthroughDetector(handler)
}

// FallthroughDetector returns a Detector whose Test function always returns
// true.  No bytes are needed for tautology.
func FallthroughDetector(hndl Handler) Detector {
	return Detector{
		Needed:  0,
		Test:    func([]byte) bool { return true },
		Handler: hndl,
	}
}

// PrefixDetector detects a static string prefix.
func PrefixDetector(prefix string, handler Handler) Detector {
	return Detector{
		Needed:  len([]byte(prefix)),
		Test:    func(b []byte) bool { return string(b) == prefix },
		Handler: handler,
	}
}

// PrefixBytesDetector detects a static string prefix.
func PrefixBytesDetector(prefix []byte, handler Handler) Detector {
	return Detector{
		Needed: len(prefix),
		Test: func(b []byte) bool {
			for i, v := range prefix {
				if b[i] != v {
					return false
				}
			}
			return true
		},
		Handler: handler,
	}
}
