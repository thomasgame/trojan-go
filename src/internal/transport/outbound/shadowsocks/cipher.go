package shadowsocks

import (
	"crypto/md5"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/shadowsocks/go-shadowsocks2/shadowaead"
)

type streamConnCipher interface {
	StreamConn(net.Conn) net.Conn
}

type aeadCipher struct {
	shadowaead.Cipher
}

func (c *aeadCipher) StreamConn(conn net.Conn) net.Conn {
	return &streamConn{
		Conn:   conn,
		Cipher: c.Cipher,
	}
}

type dummyCipher struct{}

func (dummyCipher) StreamConn(conn net.Conn) net.Conn {
	return conn
}

type streamConn struct {
	net.Conn
	shadowaead.Cipher
	reader io.Reader
	writer io.Writer
}

func (c *streamConn) initReader() error {
	salt := make([]byte, c.SaltSize())
	if _, err := io.ReadFull(c.Conn, salt); err != nil {
		return err
	}
	if inboundSaltFilter.Seen(salt) {
		return shadowaead.ErrRepeatedSalt
	}
	aead, err := c.Decrypter(salt)
	if err != nil {
		return err
	}
	c.reader = shadowaead.NewReader(c.Conn, aead)
	return nil
}

func (c *streamConn) Read(b []byte) (int, error) {
	if c.reader == nil {
		if err := c.initReader(); err != nil {
			return 0, err
		}
	}
	return c.reader.Read(b)
}

func (c *streamConn) initWriter() error {
	salt := make([]byte, c.SaltSize())
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}
	aead, err := c.Encrypter(salt)
	if err != nil {
		return err
	}
	if _, err := c.Conn.Write(salt); err != nil {
		return err
	}
	c.writer = shadowaead.NewWriter(c.Conn, aead)
	return nil
}

func (c *streamConn) Write(b []byte) (int, error) {
	if c.writer == nil {
		if err := c.initWriter(); err != nil {
			return 0, err
		}
	}
	return c.writer.Write(b)
}

type replayFilter struct {
	mu      sync.Mutex
	entries map[string]struct{}
	order   []string
	next    int
}

func newReplayFilter(size int) *replayFilter {
	return &replayFilter{
		entries: make(map[string]struct{}, size),
		order:   make([]string, 0, size),
	}
}

func (f *replayFilter) Seen(salt []byte) bool {
	key := string(salt)

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.entries[key]; ok {
		return true
	}

	if len(f.order) < cap(f.order) {
		f.order = append(f.order, key)
	} else {
		old := f.order[f.next]
		delete(f.entries, old)
		f.order[f.next] = key
		f.next = (f.next + 1) % len(f.order)
	}
	f.entries[key] = struct{}{}
	return false
}

var inboundSaltFilter = newReplayFilter(1 << 15)

func pickStreamCipher(name string, password string) (streamConnCipher, error) {
	name = strings.ToUpper(name)

	switch name {
	case "DUMMY":
		return dummyCipher{}, nil
	case "CHACHA20-IETF-POLY1305":
		name = "AEAD_CHACHA20_POLY1305"
	case "AES-128-GCM":
		name = "AEAD_AES_128_GCM"
	case "AES-256-GCM":
		name = "AEAD_AES_256_GCM"
	}

	switch name {
	case "AEAD_AES_128_GCM":
		c, err := shadowaead.AESGCM(kdf(password, 16))
		return &aeadCipher{Cipher: c}, err
	case "AEAD_AES_256_GCM":
		c, err := shadowaead.AESGCM(kdf(password, 32))
		return &aeadCipher{Cipher: c}, err
	case "AEAD_CHACHA20_POLY1305":
		c, err := shadowaead.Chacha20Poly1305(kdf(password, 32))
		return &aeadCipher{Cipher: c}, err
	default:
		return nil, errors.New("cipher not supported")
	}
}

func kdf(password string, keyLen int) []byte {
	var b, prev []byte
	h := md5.New()
	for len(b) < keyLen {
		h.Write(prev)
		h.Write([]byte(password))
		b = h.Sum(b)
		prev = b[len(b)-h.Size():]
		h.Reset()
	}
	return b[:keyLen]
}
