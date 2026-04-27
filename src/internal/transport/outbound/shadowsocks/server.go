package shadowsocks

import (
	"context"
	"net"

	"github.com/thomasgame/trojan-go/common"
	"github.com/thomasgame/trojan-go/internal/core/config"
	"github.com/thomasgame/trojan-go/internal/core/relay/redirector"
	"github.com/thomasgame/trojan-go/internal/core/tunnel"
	"github.com/thomasgame/trojan-go/internal/infra/log"
)

type Server struct {
	streamConnCipher
	*redirector.Redirector
	underlay  tunnel.Server
	redirAddr net.Addr
}

func (s *Server) AcceptConn(overlay tunnel.Tunnel) (tunnel.Conn, error) {
	conn, err := s.underlay.AcceptConn(&Tunnel{})
	if err != nil {
		return nil, common.NewError("shadowsocks failed to accept connection from underlying tunnel").Base(err)
	}
	rewindConn := common.NewRewindConn(conn)
	rewindConn.SetBufferSize(1024)
	defer rewindConn.StopBuffering()

	// try to read something from this connection
	buf := [1024]byte{}
	testConn := s.streamConnCipher.StreamConn(rewindConn)
	n, err := testConn.Read(buf[:])
	if err != nil {
		// we are under attack
		log.Error(common.NewError("shadowsocks failed to decrypt").Base(err))
		rewindConn.Rewind()
		rewindConn.StopBuffering()
		s.Redirect(&redirector.Redirection{
			RedirectTo:  s.redirAddr,
			InboundConn: rewindConn,
		})
		return nil, common.NewError("invalid aead payload")
	}
	rewindConn.StopBuffering()

	return &Conn{
		// Reuse the probed AEAD stream to avoid decrypting the same salt twice.
		aeadConn: testConn,
		Conn:     conn,
		prefix:   append([]byte(nil), buf[:n]...),
	}, nil
}

func (s *Server) AcceptPacket(t tunnel.Tunnel) (tunnel.PacketConn, error) {
	panic("not supported")
}

func (s *Server) Close() error {
	return s.underlay.Close()
}

func NewServer(ctx context.Context, underlay tunnel.Server) (*Server, error) {
	cfg := config.FromContext(ctx, Name).(*Config)
	cipher, err := pickStreamCipher(cfg.Shadowsocks.Method, cfg.Shadowsocks.Password)
	if err != nil {
		return nil, common.NewError("invalid shadowsocks cipher").Base(err)
	}
	if cfg.RemoteHost == "" {
		return nil, common.NewError("invalid shadowsocks redirection address")
	}
	if cfg.RemotePort == 0 {
		return nil, common.NewError("invalid shadowsocks redirection port")
	}
	log.Debug("shadowsocks client created")
	return &Server{
		underlay:         underlay,
		streamConnCipher: cipher,
		Redirector:       redirector.NewRedirector(ctx),
		redirAddr:        tunnel.NewAddressFromHostPort("tcp", cfg.RemoteHost, cfg.RemotePort),
	}, nil
}
