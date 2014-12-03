package proxytest

import (
	"io"
	"net"
	"testing"

	"github.com/getlantern/testify/assert"
)

var (
	ping = []byte("ping")
	pong = []byte("pong")
)

// Proxy is an interface for anything that acts like a proxy.
type Proxy interface {
	// Dial: function that dials a given destination using the proxy.
	Dial(network, addr string) (net.Conn, error)

	// Close: closes the proxy and any underlying resources
	Close() error
}

// Test tests a proxy.
func Test(t *testing.T, proxy Proxy) {
	// Set up listener for server endpoint
	sl, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Unable to listen: %s", err)
	}

	// Server that responds to ping
	go func() {
		conn, err := sl.Accept()
		if err != nil {
			t.Fatalf("Unable to accept connection: %s", err)
			return
		}
		defer conn.Close()
		b := make([]byte, 4)
		_, err = io.ReadFull(conn, b)
		if err != nil {
			t.Fatalf("Unable to read from client: %s", err)
		}
		assert.Equal(t, ping, b, "Didn't receive correct ping message")
		_, err = conn.Write(pong)
		if err != nil {
			t.Fatalf("Unable to write to client: %s", err)
		}
	}()

	conn, err := proxy.Dial(sl.Addr().Network(), sl.Addr().String())
	if err != nil {
		t.Fatalf("Unable to dial via proxy: %s", err)
	}
	defer conn.Close()

	_, err = conn.Write(ping)
	if err != nil {
		t.Fatalf("Unable to write to server via proxy: %s", err)
	}

	b := make([]byte, 4)
	_, err = io.ReadFull(conn, b)
	if err != nil {
		t.Fatalf("Unable to read from server: %s", err)
	}
	assert.Equal(t, pong, b, "Didn't receive correct pong message")
}
