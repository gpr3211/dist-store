package p2p

import (
	"net"
	"testing"
	"time"

	"github.com/gpr3211/dist-store/assert"
)

func TestSecureHandshake(t *testing.T) {
	// Create a listener for testing
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer listener.Close()

	// Channel to signal completion
	done := make(chan bool)
	var serverConn net.Conn
	var clientConn net.Conn

	// Server side goroutine
	go func() {
		conn, err := listener.Accept()
		assert.NoError(t, err)

		serverPeer := NewTCPPeer(conn, false)
		secureConn, err := SecureHandshake(serverPeer)
		assert.NoError(t, err)

		serverConn = secureConn
		done <- true
	}()

	// Client side
	conn, err := net.Dial("tcp", listener.Addr().String())
	assert.NoError(t, err)

	clientPeer := NewTCPPeer(conn, true)
	secureConn, err := SecureHandshake(clientPeer)
	assert.NoError(t, err)
	clientConn = secureConn

	// Wait for server to complete handshake
	<-done

	// Test that we got SecureConn instances back
	assert.NotNil(t, serverConn)
	assert.NotNil(t, clientConn)

	// Test encrypted communication
	testMessage := []byte("Hello, secure world!")

	// Write from client
	n, err := clientConn.Write(testMessage)
	assert.NoError(t, err)
	assert.Equal(t, len(testMessage), n)

	// Read on server
	buf := make([]byte, 1024)
	n, err = serverConn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, testMessage, buf[:n])

	// Test bidirectional communication
	responseMessage := []byte("Response from server")
	n, err = serverConn.Write(responseMessage)
	assert.NoError(t, err)

	buf = make([]byte, 1024)
	n, err = clientConn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, responseMessage, buf[:n])

	// Cleanup
	clientConn.Close()
	serverConn.Close()
}

func TestSecureHandshakeFailure(t *testing.T) {
	// Test with a closed connection
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer listener.Close()

	go func() {
		conn, _ := listener.Accept()
		conn.Close() // Close immediately
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	assert.NoError(t, err)

	// Wait a bit for server to close
	time.Sleep(100 * time.Millisecond)

	peer := NewTCPPeer(conn, true)
	_, err = SecureHandshake(peer)
	if err == nil {
		t.Error("Should have failed the handshake")
	}

	// Should fail because connection is closed.
}

func TestSecureConnReadWrite(t *testing.T) {
	// Create two peers with established connection
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer listener.Close()

	var server, client *SecureConn
	done := make(chan bool)

	// Server
	go func() {
		conn, _ := listener.Accept()
		peer := NewTCPPeer(conn, false)
		secConn, err := SecureHandshake(peer)
		assert.NoError(t, err)
		server = secConn.(*SecureConn)
		done <- true
	}()

	// Client
	conn, err := net.Dial("tcp", listener.Addr().String())
	assert.NoError(t, err)
	peer := NewTCPPeer(conn, true)
	secConn, err := SecureHandshake(peer)
	assert.NoError(t, err)
	client = secConn.(*SecureConn)

	<-done

	messages := []string{
		"First message",
		"Second message",
		"Third message with more data",
	}

	for _, msg := range messages {
		// Client -> Server
		_, err := client.Write([]byte(msg))
		assert.NoError(t, err)

		buf := make([]byte, 1024)
		n, err := server.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, msg, string(buf[:n]))
	}

	client.Close()
	server.Close()
}
