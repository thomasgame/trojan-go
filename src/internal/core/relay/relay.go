package relay

import (
	"io"
	"net"
)

// BidirectionalCopy relays data between both ends until one side returns.
func BidirectionalCopy(left, right net.Conn) error {
	errCh := make(chan error, 2)
	copyConn := func(dst, src net.Conn) {
		_, err := io.Copy(dst, src)
		errCh <- err
	}
	go copyConn(left, right)
	go copyConn(right, left)
	err := <-errCh
	if err == io.EOF {
		return nil
	}
	return err
}
