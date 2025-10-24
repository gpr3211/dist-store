package p2p

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// HybridSecureConn wraps a net.Conn with AES-GCM using a RSA handshake.
// Call HybridHandshake(conn, outbound) to create one side of the secured connection.
type HybridSecureConn struct {
	net.Conn
	priv    *rsa.PrivateKey
	peerPub *rsa.PublicKey
	aesKey  []byte      // shared AES key
	gcm     cipher.AEAD // AES-GCM

	r *bufio.Reader
	w *bufio.Writer

	writeMu sync.Mutex
	readMu  sync.Mutex

	noncePrefix [4]byte
	nonce       uint64
	nonceMu     sync.Mutex
}

// HybridHandshake performs an RSA key-exchange and establishes AES-GCM session.
// outbound should be true for the side that "initiates" (generates the AES key).
func hybridHandshakeInternal(conn net.Conn, outbound bool) (*HybridSecureConn, error) {

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate rsa: %w", err)
	}
	pub := &priv.PublicKey

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	// helper: send and receive length-prefixed byte slices
	sendBytes := func(b []byte) error {
		lenBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBuf, uint32(len(b)))
		if _, err := w.Write(lenBuf); err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
		return w.Flush()
	}

	readBytes := func() ([]byte, error) {
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			return nil, err
		}
		n := binary.BigEndian.Uint32(lenBuf)
		if n == 0 {
			return nil, nil
		}
		b := make([]byte, n)
		if _, err := io.ReadFull(r, b); err != nil {
			return nil, err
		}
		return b, nil
	}

	// 1) Exchange RSA public keys (ASN.1 / PKIX format) to be interoperable
	myPubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, ErrHandshakeInvalid
	}
	if err := sendBytes(myPubBytes); err != nil {
		return nil, ErrHandshakeInvalid
	}

	peerPubBytes, err := readBytes()
	if err != nil {
		return nil, fmt.Errorf("recv public key: %w", err)
	}
	peerIface, err := x509.ParsePKIXPublicKey(peerPubBytes)
	if err != nil {
		return nil, fmt.Errorf("parse peer public key: %w", err)
	}
	peerPub, ok := peerIface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("peer public key not RSA")
	}

	// log short fingerprint to help manually verify / detect MITM
	//	log.Printf("peer pub fingerprint=%s", fingerprint(peerPub)) TODO:

	var aesKey []byte

	if outbound {
		// Initiator: generate AES key, encrypt with peer RSA pub and send
		aesKey = make([]byte, 32) // AES-256
		if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
			return nil, fmt.Errorf("generate aes key: %w", err)
		}

		encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, peerPub, aesKey, nil)
		if err != nil {
			return nil, fmt.Errorf("encrypt aes key: %w", err)
		}
		if err := sendBytes(encryptedKey); err != nil {
			return nil, fmt.Errorf("send encrypted key: %w", err)
		}

		// Wait for responder's confirmation hash and compare
		peerHashBytes, err := readBytes()
		if err != nil {
			return nil, fmt.Errorf("read peer hash: %w", err)
		}

		myHash := sha256.Sum256(aesKey)
		if !bytes.Equal(peerHashBytes, myHash[:]) {
			log.Println("WARNING POSSIBLE MITM ATTACK !")
			//TODO:
			// add to logger.
			return nil, ErrHandshakeHashMismatch
		}
	} else {
		// Responder: receive encrypted key, decrypt, then send confirmation hash
		encryptedKey, err := readBytes()
		if err != nil {
			return nil, fmt.Errorf("recv encrypted key: %w", err)
		}
		aesKey, err = rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encryptedKey, nil)
		if err != nil {
			return nil, fmt.Errorf("decrypt aes key: %w", err)
		}

		// send confirmation hash
		h := sha256.Sum256(aesKey)
		if err := sendBytes(h[:]); err != nil {
			return nil, fmt.Errorf("send confirmation: %w", err)
		}
	}

	// 4) Create AES-GCM cipher
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	h := &HybridSecureConn{
		Conn:    conn,
		priv:    priv,
		peerPub: peerPub,
		aesKey:  aesKey,
		gcm:     gcm,
		r:       r,
		w:       w,
	}

	// randomize nonce prefix and counter to avoid reuse across restarts.
	if _, err := io.ReadFull(rand.Reader, h.noncePrefix[:]); err != nil {
		return nil, fmt.Errorf("generate nonce prefix: %w", err)
	}
	var ctrBytes [8]byte
	if _, err := io.ReadFull(rand.Reader, ctrBytes[:]); err == nil {
		h.nonce = binary.BigEndian.Uint64(ctrBytes[:])
	}

	log.Printf("hybrid handshake OK remote=%s", conn.RemoteAddr())
	return h, nil
}

// Backwards-compatible wrapper that keeps the original signature:
func HybridHandshake(conn Peer) (net.Conn, error) {
	// ensure the Peer implements net.Conn.
	rawConn, ok := conn.(net.Conn)
	if !ok {
		return nil, fmt.Errorf("HybridHandshake: conn does not implement net.Conn")
	}

	timeout := 1 * time.Second // set in config ?
	if err := rawConn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, ErrHandshakeInvalid
	}
	// check if outbound
	outbound := false
	if tcp, ok := conn.(*TCPPeer); ok {
		outbound = tcp.outbound
	}

	h, err := hybridHandshakeInternal(rawConn, outbound)
	rawConn.SetDeadline(time.Time{}) // remove deadline after handshake is done

	return h, err
}

// Write encrypts and sends data as :
//
//	-- [len][nonce||ciphertext]
func (h *HybridSecureConn) Write(p []byte) (int, error) {
	h.writeMu.Lock()
	defer h.writeMu.Unlock()

	nonce := h.getNextNonce()
	ciphertext := h.gcm.Seal(nil, nonce, p, nil)

	msgLen := uint32(len(nonce) + len(ciphertext))
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, msgLen)

	if _, err := h.w.Write(lenBuf); err != nil {
		return 0, err
	}
	if _, err := h.w.Write(nonce); err != nil {
		return 0, err
	}
	if _, err := h.w.Write(ciphertext); err != nil {
		return 0, err
	}
	if err := h.w.Flush(); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Read reads a framed message and decrypts it into p
func (h *HybridSecureConn) Read(p []byte) (int, error) {
	h.readMu.Lock()
	defer h.readMu.Unlock()

	// read length prefix
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(h.r, lenBuf); err != nil {
		return 0, err
	}
	msgLen := binary.BigEndian.Uint32(lenBuf)
	if msgLen == 0 {
		return 0, nil
	}
	msg := make([]byte, msgLen)
	if _, err := io.ReadFull(h.r, msg); err != nil {
		return 0, err
	}

	nonceSize := h.gcm.NonceSize()
	if len(msg) < nonceSize {
		return 0, fmt.Errorf("message too short")
	}
	nonce := msg[:nonceSize]
	ciphertext := msg[nonceSize:]

	plaintext, err := h.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return 0, fmt.Errorf("decrypt failed: %w", err)
	}

	n := copy(p, plaintext)
	if n < len(plaintext) {
		return n, io.ErrShortBuffer
	}
	return n, nil
}

// getNextNonce builds a 12-byte nonce: 4 bytes prefix + 8 bytes counter
func (h *HybridSecureConn) getNextNonce() []byte {
	h.nonceMu.Lock()
	defer h.nonceMu.Unlock()

	nonce := make([]byte, 12)
	copy(nonce[:4], h.noncePrefix[:])
	binary.BigEndian.PutUint64(nonce[4:], h.nonce)
	h.nonce++
	return nonce
}

// Close zeroes the AES key and closes underlying connection
func (h *HybridSecureConn) Close() error {
	if h.aesKey != nil {
		for i := range h.aesKey {
			h.aesKey[i] = 0
		}
		h.aesKey = nil
	}
	return h.Conn.Close()
}

// fingerprint returns a short hex fingerprint of an RSA public key
func fingerprint(pub *rsa.PublicKey) string {
	h := sha256.Sum256(pub.N.Bytes())
	return hex.EncodeToString(h[:8])
}
