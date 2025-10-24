package p2p

import (
	"bytes"
	"encoding/gob"
	"net"
	"testing"
	"time"

	"github.com/gpr3211/dist-store/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	metadata := model.Message{
		Payload: model.MessageStoreFile{
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

		var meta model.Message
		gob.NewDecoder(bytes.NewReader(metaBuf[:n])).Decode(&meta)
		receivedMeta := meta.Payload.(model.MessageStoreFile)

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
	//	time.Sleep(100 * time.Millisecond)

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

	// Wait for receiver
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
	t.Logf("✓ HYBRID Throughput: %.2f MB/s", throughput)

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
	t.Log("Testing Bi-Directional data transfer")

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

	<-done

	client.Close()
	server.Close()
}

// Benchmar.k RSA encryption (small messages only)
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

		buf := make([]byte, 2*1024*1024) // 2MB
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

func TestRSAvsHybridComparison(t *testing.T) {
	t.Log("=== RSA vs Hybrid Performance Comparison ===")

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
