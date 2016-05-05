# Go Stacked Servers

Stacked implements a multi-protocol server as a stack of heuristic detectors.

A detector is simply the combination of a test function and a connection
handler.

A convenience layer is provided for integrating net.Listener oriented servers,
such as net/http.

# Documentation

See the [godoc](https://godoc.org/github.com/uber-common/stacked).
