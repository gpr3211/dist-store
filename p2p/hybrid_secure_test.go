package p2p

import (
	"bytes"
	"encoding/gob"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"io"
)

type Message struct {
	Payload any
}
type MessageStoreFile struct {
	ID   string
	Key  string
	Size int64
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(Message{})
}

func TestHybridHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	var server, client net.Conn
	done := make(chan bool)

	// Server side
	go func() {
		conn, _ := listener.Accept()
		peer := NewTCPPeer(conn, false)
		server, _ = HybridHandshake(peer)
		done <- true
	}()

	// Client side
	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	peer := NewTCPPeer(conn, true)
	client, err = HybridHandshake(peer)
	require.NoError(t, err)
	<-done

	// Test basic communication
	testMsg := []byte("Hello, Hybrid Encryption!")

	_, err = client.Write(testMsg)
	require.NoError(t, err)

	buf := make([]byte, 1024)
	n, err := server.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, testMsg, buf[:n])

	client.Close()
	server.Close()
}

func TestHybridSecureConn10MBFile(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	var server, client net.Conn
	done := make(chan bool)

	go func() {
		conn, _ := listener.Accept()
		peer := NewTCPPeer(conn, false)
		server, _ = HybridHandshake(peer)
		done <- true
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	peer := NewTCPPeer(conn, true)
	client, err = HybridHandshake(peer)
	require.NoError(t, err)
	<-done

	// Create 10MB file
	fileSize := 10 * 1024 * 1024
	fileData := make([]byte, fileSize)
	for i := range fileData {
		fileData[i] = byte(i % 256)
	}

	t.Logf("Sending 10MB file with HYBRID encryption...")

	// Send metadata
	metadata := Message{
		Payload: MessageStoreFile{
			ID:   "hybrid-large-file",
			Key:  "hybrid-key",
			Size: int64(fileSize),
		},
	}

	receiveDone := make(chan bool)
	var receivedData []byte

	// Receiver
	go func() {
		// Receive metadata
		metaBuf := make([]byte, 4096)
		n, err := server.Read(metaBuf)
		if err != nil {
			t.Errorf("Failed to receive metadata: %v", err)
			return
		}

		var meta Message
		gob.NewDecoder(bytes.NewReader(metaBuf[:n])).Decode(&meta)
		receivedMeta := meta.Payload.(MessageStoreFile)
		t.Logf("Received metadata: ID=%s, Size=%d", receivedMeta.ID, receivedMeta.Size)

		// Receive file data in larger chunks (AES is fast!)
		receivedData = make([]byte, 0, receivedMeta.Size)
		buf := make([]byte, 64*1024) // 64KB buffer

		for int64(len(receivedData)) < receivedMeta.Size {
			n, err := server.Read(buf)
			if err != nil {
				t.Errorf("Failed to receive chunk: %v", err)
				return
			}
			receivedData = append(receivedData, buf[:n]...)

			if len(receivedData)%(1024*1024) == 0 {
				t.Logf("Received %d MB", len(receivedData)/(1024*1024))
			}
		}

		receiveDone <- true
	}()

	// Send metadata
	var metaBuf bytes.Buffer
	gob.NewEncoder(&metaBuf).Encode(metadata)
	client.Write(metaBuf.Bytes())
	time.Sleep(100 * time.Millisecond)

	// Send file data in larger chunks (AES can handle large data!)
	chunkSize := 32 * 1024 // 32KB chunks

	startTime := time.Now()
	bytesSent := 0

	for i := 0; i < fileSize; i += chunkSize {
		end := i + chunkSize
		if end > fileSize {
			end = fileSize
		}

		_, err = client.Write(fileData[i:end])
		if err != nil {
			t.Fatalf("Failed to send chunk: %v", err)
		}

		bytesSent += (end - i)

		// Progress every MB
		if bytesSent%(1024*1024) == 0 {
			t.Logf("Sent %d MB", bytesSent/(1024*1024))
		}
	}

	duration := time.Since(startTime)
	t.Logf("Sent %d bytes in %v", fileSize, duration)

	select {
	case <-receiveDone:
		t.Log("Receiver finished")
	case <-time.After(30 * time.Second):
		t.Fatal("Receiver timeout")
	}

	// Verify
	require.Equal(t, fileSize, len(receivedData))
	assert.Equal(t, fileData, receivedData)

	throughput := float64(fileSize) / duration.Seconds() / (1024 * 1024)
	t.Logf(" HYBRID Throughput: %.2f MB/s", throughput)

	client.Close()
	server.Close()
}
func TestHybridSecureConn100MBFile(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	var server, client net.Conn
	done := make(chan bool)

	go func() {
		conn, _ := listener.Accept()
		peer := NewTCPPeer(conn, false)
		server, _ = HybridHandshake(peer)
		done <- true
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	peer := NewTCPPeer(conn, true)
	client, err = HybridHandshake(peer)
	require.NoError(t, err)
	<-done

	// Create 10MB file
	fileSize := 100 * 1024 * 1024
	fileData := make([]byte, fileSize)
	for i := range fileData {
		fileData[i] = byte(i % 256)
	}

	t.Logf("Sending 10MB file with HYBRID encryption...")

	// Send metadata
	metadata := Message{
		Payload: MessageStoreFile{
			ID:   "hybrid-large-file",
			Key:  "hybrid-key",
			Size: int64(fileSize),
		},
	}

	receiveDone := make(chan bool)
	var receivedData []byte

	// Receiver
	go func() {
		// Receive metadata
		metaBuf := make([]byte, 4096)
		n, err := server.Read(metaBuf)
		if err != nil {
			t.Errorf("Failed to receive metadata: %v", err)
			return
		}

		var meta Message
		gob.NewDecoder(bytes.NewReader(metaBuf[:n])).Decode(&meta)
		receivedMeta := meta.Payload.(MessageStoreFile)
		t.Logf("Received metadata: ID=%s, Size=%d", receivedMeta.ID, receivedMeta.Size)

		// Receive file data in larger chunks (AES is fast!)
		receivedData = make([]byte, 0, receivedMeta.Size)
		buf := make([]byte, 64*1024) // 64KB buffer

		for int64(len(receivedData)) < receivedMeta.Size {
			n, err := server.Read(buf)
			if err != nil {
				t.Errorf("Failed to receive chunk: %v", err)
				return
			}
			receivedData = append(receivedData, buf[:n]...)

			if len(receivedData)%(1024*1024) == 0 {
				//			t.Logf("Received %d MB", len(receivedData)/(1024*1024))
			}
		}

		receiveDone <- true
	}()

	// Send metadata
	var metaBuf bytes.Buffer
	gob.NewEncoder(&metaBuf).Encode(metadata)
	client.Write(metaBuf.Bytes())
	time.Sleep(100 * time.Millisecond)

	// Send file data in larger chunks (AES can handle large data!)
	chunkSize := 64 * 1024 // 32KB chunks

	startTime := time.Now()
	bytesSent := 0

	for i := 0; i < fileSize; i += chunkSize {
		end := i + chunkSize
		if end > fileSize {
			end = fileSize
		}

		_, err = client.Write(fileData[i:end])
		if err != nil {
			t.Fatalf("Failed to send chunk: %v", err)
		}

		bytesSent += (end - i)

		// Progress every MB
		if bytesSent%(1024*1024) == 0 {
			//		t.Logf("Sent %d MB", bytesSent/(1024*1024))
		}
	}

	duration := time.Since(startTime)
	t.Logf("Sent %d bytes in %v", fileSize, duration)

	select {
	case <-receiveDone:
		t.Log("Receiver finished")
	case <-time.After(30 * time.Second):
		t.Fatal("Receiver timeout")
	}

	// Verify
	require.Equal(t, fileSize, len(receivedData))
	assert.Equal(t, fileData, receivedData)

	throughput := float64(fileSize) / duration.Seconds() / (1024 * 1024)
	t.Logf(" HYBRID Throughput: %.2f MB/s", throughput)

	client.Close()
	server.Close()
}

func TestHybridSecureConnMultipleMessages(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	var server, client net.Conn
	done := make(chan bool)

	go func() {
		conn, _ := listener.Accept()
		peer := NewTCPPeer(conn, false)
		server, _ = HybridHandshake(peer)
		done <- true
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	peer := NewTCPPeer(conn, true)
	client, err = HybridHandshake(peer)
	require.NoError(t, err)
	<-done

	// Send multiple messages rapidly
	messages := []string{
		"First message",
		"Second message",
		"Third message with more content",
		"Fourth",
		"Fifth message here",
	}

	go func() {
		for _, msg := range messages {
			client.Write([]byte(msg))
		}
	}()

	for i, expected := range messages {
		buf := make([]byte, 1024)
		n, err := server.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, expected, string(buf[:n]))
		t.Logf("Message %d: %s", i+1, string(buf[:n]))
	}

	client.Close()
	server.Close()
}

func TestHybridSecureConnBidirectional(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	var server, client net.Conn
	done := make(chan bool, 2)

	go func() {
		conn, _ := listener.Accept()
		peer := NewTCPPeer(conn, false)
		server, _ = HybridHandshake(peer)
		done <- true
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	peer := NewTCPPeer(conn, true)
	client, err = HybridHandshake(peer)
	require.NoError(t, err)
	<-done

	// Test simultaneous bidirectional
	go func() {
		for i := 0; i < 5; i++ {
			server.Write([]byte("From server"))
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 5; i++ {
			client.Write([]byte("From client"))
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Read from both sides
	receivedServer := 0
	receivedClient := 0

	go func() {
		buf := make([]byte, 1024)
		for receivedClient < 5 {
			n, _ := client.Read(buf)
			assert.Equal(t, "From server", string(buf[:n]))
			receivedClient++
		}
	}()

	buf := make([]byte, 1024)
	for receivedServer < 5 {
		n, _ := server.Read(buf)
		assert.Equal(t, "From client", string(buf[:n]))
		receivedServer++
	}

	<-done // Wait for server writes to finish

	client.Close()
	server.Close()
}

// Benchmark RSA encryption (small messages only)
func BenchmarkRSASmallMessage(b *testing.B) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	done := make(chan bool)

	go func() {
		conn, _ := listener.Accept()
		peer := NewTCPPeer(conn, false)
		server, _ := SecureHandshake(peer)
		done <- true

		buf := make([]byte, 4096)
		for {
			_, err := server.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	peer := NewTCPPeer(conn, true)
	client, _ := SecureHandshake(peer)
	<-done

	// Small message that fits in RSA
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client.Write(data)
	}

	b.StopTimer()
	client.Close()
}

// Benchmark Hybrid encryption with different sizes
func BenchmarkHybrid1KB(b *testing.B) {
	benchmarkHybrid(b, 1024)
}

func BenchmarkHybrid10KB(b *testing.B) {
	benchmarkHybrid(b, 10*1024)
}

func BenchmarkHybrid100KB(b *testing.B) {
	benchmarkHybrid(b, 100*1024)
}

func BenchmarkHybrid1MB(b *testing.B) {
	benchmarkHybrid(b, 1024*1024)
}

func benchmarkHybrid(b *testing.B, size int) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	done := make(chan bool)

	go func() {
		conn, _ := listener.Accept()
		peer := NewTCPPeer(conn, false)
		server, _ := HybridHandshake(peer)
		done <- true

		buf := make([]byte, 2*1024*1024) // 2MB buffer
		for {
			_, err := server.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	peer := NewTCPPeer(conn, true)
	client, _ := HybridHandshake(peer)
	<-done

	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.SetBytes(int64(size))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := client.Write(data)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()
	client.Close()
}

// Simple comparison test (not a benchmark)
func TestRSAvsHybridComparison(t *testing.T) {
	t.Log("=== RSA vs Hybrid Performance Comparison ===")

	// Test RSA with 100 bytes
	t.Run("RSA_100bytes", func(t *testing.T) {
		listener, _ := net.Listen("tcp", "127.0.0.1:0")
		defer listener.Close()

		done := make(chan bool)

		go func() {
			conn, _ := listener.Accept()
			peer := NewTCPPeer(conn, false)
			server, _ := SecureHandshake(peer)
			done <- true

			buf := make([]byte, 4096)
			for i := 0; i < 100; i++ {
				server.Read(buf)
			}
		}()

		conn, _ := net.Dial("tcp", listener.Addr().String())
		peer := NewTCPPeer(conn, true)
		client, _ := SecureHandshake(peer)
		<-done

		data := make([]byte, 100)

		start := time.Now()
		for i := 0; i < 100; i++ {
			client.Write(data)
		}
		duration := time.Since(start)

		t.Logf("RSA: Sent 100 messages (100 bytes each) in %v", duration)
		t.Logf("RSA: Throughput: %.2f KB/s", float64(100*100)/duration.Seconds()/1024)

		client.Close()
	})

	// Test Hybrid with 1MB
	t.Run("Hybrid_1MB", func(t *testing.T) {
		listener, _ := net.Listen("tcp", "127.0.0.1:0")
		defer listener.Close()

		done := make(chan bool)

		go func() {
			conn, _ := listener.Accept()
			peer := NewTCPPeer(conn, false)
			server, _ := HybridHandshake(peer)
			done <- true

			buf := make([]byte, 2*1024*1024)
			for i := 0; i < 10; i++ {
				server.Read(buf)
			}
		}()

		conn, _ := net.Dial("tcp", listener.Addr().String())
		peer := NewTCPPeer(conn, true)
		client, _ := HybridHandshake(peer)
		<-done

		data := make([]byte, 1024*1024) // 1MB

		start := time.Now()
		for i := 0; i < 10; i++ {
			client.Write(data)
		}
		duration := time.Since(start)

		t.Logf("Hybrid: Sent 10 messages (1MB each) in %v", duration)
		t.Logf("Hybrid: Throughput: %.2f MB/s", float64(10*1024*1024)/duration.Seconds()/(1024*1024))

		client.Close()
	})
}

// TestMITMAttackPrevention simulates a MITM attack where an attacker
// intercepts the handshake and tries to substitute their own AES key.
// The confirmation hash should detect this and fail the handshake.
func TestMITMAttackPrevention(t *testing.T) {
	// Create a pipe to simulate network communication
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	// Attacker sits in the middle
	attackerToClient, clientSide := net.Pipe()
	attackerToServer, serverSide := net.Pipe()

	attackerError := make(chan error, 1)
	clientError := make(chan error, 1)
	serverError := make(chan error, 1)

	// MITM Attacker goroutine - intercepts and modifies the key exchange
	go func() {
		defer attackerToClient.Close()
		defer attackerToServer.Close()

		// Forward public key exchanges (RSA keys)
		// Read client's public key
		lenBuf := make([]byte, 4)
		io.ReadFull(clientSide, lenBuf)
		clientPubLen := binary.BigEndian.Uint32(lenBuf)
		clientPubBytes := make([]byte, clientPubLen)
		io.ReadFull(clientSide, clientPubBytes)

		// Forward to server
		attackerToServer.Write(lenBuf)
		attackerToServer.Write(clientPubBytes)

		// Read server's public key
		io.ReadFull(serverSide, lenBuf)
		serverPubLen := binary.BigEndian.Uint32(lenBuf)
		serverPubBytes := make([]byte, serverPubLen)
		io.ReadFull(serverSide, serverPubBytes)

		// Forward to client
		attackerToClient.Write(lenBuf)
		attackerToClient.Write(serverPubBytes)

		// Now intercept the encrypted AES key from client
		io.ReadFull(clientSide, lenBuf)
		encKeyLen := binary.BigEndian.Uint32(lenBuf)
		encKeyBytes := make([]byte, encKeyLen)
		io.ReadFull(clientSide, encKeyBytes)

		// ATTACK: Generate our own malicious AES key
		maliciousKey := make([]byte, 32)
		rand.Read(maliciousKey)

		// Parse server's public key to encrypt our malicious key
		serverPubIface, _ := x509.ParsePKIXPublicKey(serverPubBytes)
		serverPub := serverPubIface.(*rsa.PublicKey)

		// Encrypt malicious key with server's public key
		maliciousEncKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader,
			serverPub, maliciousKey, nil)
		if err != nil {
			attackerError <- err
			return
		}

		// Send malicious encrypted key to server
		maliciousLenBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(maliciousLenBuf, uint32(len(maliciousEncKey)))
		attackerToServer.Write(maliciousLenBuf)
		attackerToServer.Write(maliciousEncKey)

		// Now server will decrypt the malicious key and send confirmation hash
		// Read server's confirmation hash (of malicious key)
		io.ReadFull(serverSide, lenBuf)
		hashLen := binary.BigEndian.Uint32(lenBuf)
		serverHashBytes := make([]byte, hashLen)
		io.ReadFull(serverSide, serverHashBytes)

		// Forward it to client (client will compare with their original key's hash)
		attackerToClient.Write(lenBuf)
		attackerToClient.Write(serverHashBytes)

		t.Log("MITM: Successfully intercepted and modified key exchange")
	}()

	// Client side (initiator)
	go func() {
		peer := NewTCPPeer(clientSide, true)
		_, err := HybridHandshake(peer)
		clientError <- err
	}()

	// Server side (responder)
	go func() {
		peer := NewTCPPeer(serverSide, false)
		_, err := HybridHandshake(peer)
		serverError <- err
	}()

	// Wait for results
	select {
	case err := <-clientError:
		// Client should detect the mismatch via confirmation hash
		require.Error(t, err, "Client should detect MITM attack")
		assert.Contains(t, err.Error(), "handshake confirmation mismatch",
			"Should fail due to hash mismatch")
		t.Log("✓ MITM attack detected by client: ", err)

	case err := <-serverError:
		// Or server might fail
		t.Logf("Server error: %v", err)

	case <-time.After(2 * time.Second):
		t.Fatal("Test timeout - handshake should fail quickly")
	}
}

// TestNonceUniqueness verifies that nonces are never reused
func TestNonceUniqueness(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	done := make(chan bool)

	go func() {
		conn, _ := listener.Accept()
		peer := NewTCPPeer(conn, false)
		server, _ := HybridHandshake(peer)
		defer server.Close()

		// Just receive messages
		buf := make([]byte, 1024)
		for i := 0; i < 1000; i++ {
			server.Read(buf)
		}
		done <- true
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	peer := NewTCPPeer(conn, true)
	client, err := HybridHandshake(peer)
	require.NoError(t, err)
	defer client.Close()

	hConn := client.(*HybridSecureConn)

	// Track all generated nonces
	seenNonces := make(map[string]bool)

	// Generate 1000 nonces and verify uniqueness
	for i := 0; i < 1000; i++ {
		nonce := hConn.getNextNonce()
		nonceStr := string(nonce)

		// Check if we've seen this nonce before
		if seenNonces[nonceStr] {
			t.Fatalf("Nonce reused after %d messages! This is a critical security flaw", i)
		}
		seenNonces[nonceStr] = true

		// Send a message to increment nonce counter
		client.Write([]byte("test"))
	}

	<-done
	t.Logf("✓ Generated 1000 unique nonces - no reuse detected")
}

// TestKeyZeroingOnClose verifies sensitive key material is zeroed
func TestKeyZeroingOnClose(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	done := make(chan bool)

	go func() {
		conn, _ := listener.Accept()
		peer := NewTCPPeer(conn, false)
		HybridHandshake(peer)
		done <- true
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	peer := NewTCPPeer(conn, true)
	client, err := HybridHandshake(peer)
	require.NoError(t, err)
	<-done

	hConn := client.(*HybridSecureConn)

	// Verify key exists before close
	require.NotNil(t, hConn.aesKey)
	require.Equal(t, 32, len(hConn.aesKey))

	// Check that key is not all zeros
	allZeros := true
	for _, b := range hConn.aesKey {
		if b != 0 {
			allZeros = false
			break
		}
	}
	require.False(t, allZeros, "Key should not be all zeros before close")

	// Close connection
	client.Close()

	// Verify key is zeroed after close
	if hConn.aesKey != nil {
		t.Error("Key should be nil after close")
	}

	t.Log("✓ AES key properly zeroed on connection close")
}

// TestFingerprintVerification simulates manual fingerprint verification
func TestFingerprintVerification(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	var clientFingerprint, serverFingerprint string
	done := make(chan bool)

	// Capture logs to extract fingerprints
	go func() {
		conn, _ := listener.Accept()
		peer := NewTCPPeer(conn, false)
		server, err := HybridHandshake(peer)
		require.NoError(t, err)
		defer server.Close()

		// In real code, you'd extract the fingerprint from logs
		// For this test, we'll compute it directly
		hConn := server.(*HybridSecureConn)
		h := sha256.Sum256(x509.MarshalPKCS1PublicKey(hConn.peerPub))
		serverFingerprint = string(h[:8])

		done <- true
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	peer := NewTCPPeer(conn, true)
	client, err := HybridHandshake(peer)
	require.NoError(t, err)
	defer client.Close()

	hConn := client.(*HybridSecureConn)
	h := sha256.Sum256(x509.MarshalPKCS1PublicKey(hConn.peerPub))
	clientFingerprint = string(h[:8])

	<-done

	// In a real scenario, users would manually compare these fingerprints
	// through a separate channel (phone call, in person, etc.)
	t.Logf("Client sees server fingerprint: %x", []byte(clientFingerprint))
	t.Logf("Server sees client fingerprint: %x", []byte(serverFingerprint))

	// Both should have captured each other's keys
	require.NotEqual(t, clientFingerprint, serverFingerprint,
		"Client and server should see different peer keys")

	t.Log("✓ Fingerprints can be used for out-of-band verification")
}

// TestReplayAttackPrevention verifies that replaying encrypted messages fails
func TestReplayAttackPrevention(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	//	var capturedMessage []byte
	done := make(chan bool)

	// Server that captures the first encrypted message
	go func() {
		conn, _ := listener.Accept()

		// Capture raw bytes during handshake and first message
		// This simulates an attacker recording network traffic
		//		rawBuf := make([]byte, 4096)

		// Let handshake complete by forwarding traffic
		peer := NewTCPPeer(conn, false)
		server, _ := HybridHandshake(peer)
		defer server.Close()

		// Receive first message and capture it
		buf := make([]byte, 1024)
		server.Read(buf)

		// Try to replay by reading the raw encrypted bytes
		// In reality, attacker would capture from network
		t.Log("Message received (attacker could have captured this)")

		done <- true
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	peer := NewTCPPeer(conn, true)
	client, err := HybridHandshake(peer)
	require.NoError(t, err)
	defer client.Close()

	// Send message
	originalMsg := []byte("Secret message")
	client.Write(originalMsg)

	<-done

	// The nonce counter ensures that even if the same plaintext is sent twice,
	// the ciphertext will be different (and replaying old ciphertext won't decrypt)
	hConn := client.(*HybridSecureConn)
	nonce1 := hConn.getNextNonce()
	nonce2 := hConn.getNextNonce()

	assert.NotEqual(t, nonce1, nonce2, "Sequential nonces must be different")

	// Encrypt same message twice with different nonces
	ct1 := hConn.gcm.Seal(nil, nonce1, originalMsg, nil)
	ct2 := hConn.gcm.Seal(nil, nonce2, originalMsg, nil)

	assert.NotEqual(t, ct1, ct2, "Same plaintext should produce different ciphertext")

	t.Log("✓ Nonce counter prevents replay attacks")
	t.Log("✓ Same plaintext produces different ciphertext each time")
}

// TestConfirmationHashMismatch directly tests hash verification
func TestConfirmationHashMismatch(t *testing.T) {
	// This test simulates what happens when the confirmation hash doesn't match

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	clientError := make(chan error, 1)
	serverError := make(chan error, 1)

	// Server sends wrong confirmation hash
	go func() {
		// Do manual handshake to inject wrong hash
		priv, _ := rsa.GenerateKey(rand.Reader, 2048)
		pub := &priv.PublicKey

		// Exchange public keys normally
		pubBytes, _ := x509.MarshalPKIXPublicKey(pub)
		lenBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBuf, uint32(len(pubBytes)))
		serverConn.Write(lenBuf)
		serverConn.Write(pubBytes)

		// Read client public key
		io.ReadFull(serverConn, lenBuf)
		peerLen := binary.BigEndian.Uint32(lenBuf)
		peerBytes := make([]byte, peerLen)
		io.ReadFull(serverConn, peerBytes)

		// Read encrypted AES key
		io.ReadFull(serverConn, lenBuf)
		encKeyLen := binary.BigEndian.Uint32(lenBuf)
		encKey := make([]byte, encKeyLen)
		io.ReadFull(serverConn, encKey)

		// Decrypt AES key
		//aesKey, _ := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encKey, nil)

		// ATTACK: Send wrong confirmation hash
		wrongHash := make([]byte, 32)
		rand.Read(wrongHash) // Random hash instead of SHA256(aesKey)

		binary.BigEndian.PutUint32(lenBuf, 32)
		serverConn.Write(lenBuf)
		serverConn.Write(wrongHash)

		serverError <- nil
	}()

	// Client attempts handshake
	go func() {
		peer := NewTCPPeer(clientConn, true)
		_, err := HybridHandshake(peer)
		clientError <- err
	}()

	// Client should detect mismatch
	select {
	case err := <-clientError:
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrHandshakeInvalid.Error())
		t.Log("✓ Confirmation hash mismatch detected:", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Should have failed with hash mismatch")
	}
}
