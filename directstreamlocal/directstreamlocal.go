package directstreamlocal

import (
	"context"
	"net"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

// DirectStreamLocal Handling for a single incoming connection
type DirectStreamLocal struct {
	NewChan ssh.NewChannel
	*sshd.ServerConn
}

func (s *DirectStreamLocal) Handle(ctx context.Context) {
	var msg sshd.StreamLocalChannelOpenDirectMsg
	if err := ssh.Unmarshal(s.NewChan.ExtraData(), &msg); err != nil {
		if s.Logger != nil {
			s.Logger.Println("unable to setup forwarding:", err)
		}
		s.NewChan.Reject(ssh.ResourceShortage, "Error parsing message")
		return
	}

	outbound, err := s.proxyDial(ctx, "unix", msg.SocketPath)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Println("unable to dial forward:", err)
		}
		s.NewChan.Reject(ssh.ConnectionFailed, err.Error())
		return
	}
	defer outbound.Close()

	ch, reqs, err := s.NewChan.Accept()
	if err != nil {
		if s.Logger != nil {
			s.Logger.Println("unable to accept chan:", err)
		}
		return
	}
	defer ch.Close()

	var buf1, buf2 []byte
	if s.BytesPool != nil {
		buf1 = s.BytesPool.Get()
		buf2 = s.BytesPool.Get()
		defer func() {
			s.BytesPool.Put(buf1)
			s.BytesPool.Put(buf2)
		}()
	} else {
		buf1 = make([]byte, 32*1024)
		buf2 = make([]byte, 32*1024)
	}

	go sshd.DiscardRequests(s.Logger, reqs)
	err = sshd.Tunnel(ctx, ch, outbound, buf1, buf2)
	if err != nil && !sshd.IsClosedConnError(err) {
		if s.Logger != nil {
			s.Logger.Println("Tunnel:", err)
		}
		return
	}
}

func (s *DirectStreamLocal) proxyDial(ctx context.Context, network, address string) (net.Conn, error) {
	proxyDial := s.ProxyDial
	if proxyDial == nil {
		var dialer net.Dialer
		proxyDial = dialer.DialContext
	}
	return proxyDial(ctx, network, address)
}