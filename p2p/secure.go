package p2p

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
)

// SecureConn wraps a net.Conn and transparently encrypts/decrypts traffic
type SecureConn struct {
	net.Conn
	priv *rsa.PrivateKey
	pub  *rsa.PublicKey
	mu   sync.Mutex
}

// Write encrypts data with the peer's public key before sending
func (s *SecureConn) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	enc, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, s.pub, p, nil)
	if err != nil {
		return 0, err
	}
	_, err = s.Conn.Write(enc)
	return len(p), err
}

// Read decrypts received data with our private key
func (s *SecureConn) Read(p []byte) (int, error) {
	// read full encrypted block
	buf := make([]byte, 4096)
	n, err := s.Conn.Read(buf)
	if err != nil {
		return 0, err
	}
	dec, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, s.priv, buf[:n], nil)
	if err != nil {
		return 0, err
	}
	copy(p, dec)
	return len(dec), nil
}

func (s *SecureConn) private() *rsa.PrivateKey { return s.priv }

// HandshakeFunc performs RSA key exchange and returns a *SecureConn.
func SecureHandshake(conn Peer) (net.Conn, error) {
	// 1. Generate our RSA keys
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	pub := &priv.PublicKey

	// 2. Exchange public keys.
	enc := gob.NewEncoder(conn)
	dec := gob.NewDecoder(conn)

	var peerKey rsa.PublicKey
	if err := enc.Encode(pub); err != nil {
		return nil, fmt.Errorf("send pubkey: %w", err)
	}
	if err := dec.Decode(&peerKey); err != nil {
		return nil, fmt.Errorf("recv pubkey: %w", err)
	}

	fmt.Println(" Handshake OK with", conn.RemoteAddr())

	// 3. Wrap connection
	return &SecureConn{Conn: conn, priv: priv, pub: &peerKey}, nil
}
