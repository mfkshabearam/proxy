package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/getlantern/golog"
)

var (
	log = golog.LoggerFor("proxy-intercept")
)

// Intercept intercepts an HTTP CONNECT request, hijacks the underlying client
// connetion and starts piping the data over a new net.Conn obtained from the
// given Dialer.
func Intercept(d Dialer, resp http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		panic("Intercept used for non-CONNECT request!")
	}

	// Hijack underlying connection
	clientConn, _, err := resp.(http.Hijacker).Hijack()
	if err != nil {
		resp.WriteHeader(502)
		fmt.Fprintf(resp, "Unable to hijack connection: %s", err)
		return
	}
	defer clientConn.Close()

	addr := hostIncludingPort(req, 443)

	// Establish outbound connection
	connOut, err := d.Dial("tcp", addr)
	if err != nil {
		resp.WriteHeader(502)
		fmt.Fprintf(resp, "Unable to handle CONNECT request: %s", err)
		return
	}
	defer connOut.Close()

	// Pipe data
	pipeData(clientConn, connOut, req)
}

// pipeData pipes data between the client and proxy connections.  It's also
// responsible for responding to the initial CONNECT request with a 200 OK.
func pipeData(clientConn net.Conn, connOut net.Conn, req *http.Request) {
	// Start piping to proxy
	go io.Copy(connOut, clientConn)

	// Respond OK
	err := respondOK(clientConn, req)
	if err != nil {
		log.Errorf("Unable to respond OK: %s", err)
		return
	}

	// Then start coyping from out to client
	io.Copy(clientConn, connOut)
}

func respondOK(writer io.Writer, req *http.Request) error {
	defer req.Body.Close()
	resp := &http.Response{
		StatusCode: 200,
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
	return resp.Write(writer)
}

// hostIncludingPort extracts the host:port from a request.  It fills in a
// a default port if none was found in the request.
func hostIncludingPort(req *http.Request, defaultPort int) string {
	parts := strings.Split(req.Host, ":")
	if len(parts) == 1 {
		return req.Host + ":" + strconv.Itoa(defaultPort)
	} else {
		return req.Host
	}
}
