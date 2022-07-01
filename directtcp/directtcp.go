package directtcp

import (
	"context"
	"fmt"
	"net"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

// DirectTCP Handling for a single incoming connection
type DirectTCP struct{}

func (s *DirectTCP) Handle(ctx context.Context, newChan ssh.NewChannel, serverConn *sshd.ServerConn) {
	var msg sshd.ChannelOpenDirectMsg
	if err := ssh.Unmarshal(newChan.ExtraData(), &msg); err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("unable to setup forwarding:", err)
		}
		newChan.Reject(ssh.ResourceShortage, "Error parsing message")
		return
	}

	outbound, err := s.proxyDial(ctx, serverConn, "tcp", fmt.Sprintf("%s:%d", msg.RAddr, msg.RPort))
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("unable to dial forward:", err)
		}
		newChan.Reject(ssh.ConnectionFailed, err.Error())
		return
	}
	defer outbound.Close()

	ch, reqs, err := newChan.Accept()
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("unable to accept chan:", err)
		}
		return
	}
	defer ch.Close()

	var buf1, buf2 []byte
	if serverConn.BytesPool != nil {
		buf1 = serverConn.BytesPool.Get()
		buf2 = serverConn.BytesPool.Get()
		defer func() {
			serverConn.BytesPool.Put(buf1)
			serverConn.BytesPool.Put(buf2)
		}()
	} else {
		buf1 = make([]byte, 32*1024)
		buf2 = make([]byte, 32*1024)
	}

	go sshd.DiscardRequests(serverConn.Logger, reqs)
	err = sshd.Tunnel(ctx, ch, outbound, buf1, buf2)
	if err != nil && !sshd.IsClosedConnError(err) {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("Tunnel:", err)
		}
		return
	}
}

func (s *DirectTCP) proxyDial(ctx context.Context, serverConn *sshd.ServerConn, network, address string) (net.Conn, error) {
	proxyDial := serverConn.ProxyDial
	if proxyDial == nil {
		var dialer net.Dialer
		proxyDial = dialer.DialContext
	}
	return proxyDial(ctx, network, address)
}
