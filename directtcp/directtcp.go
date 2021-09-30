package directtcp

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

// DirectTCP Handling for a single incoming connection
type DirectTCP struct {
	NewChan ssh.NewChannel
	// BytesPool getting and returning temporary bytes for use by io.CopyBuffer
	BytesPool sshd.BytesPool
	// Logger error log
	Logger sshd.Logger
	// ProxyDial specifies the optional proxyDial function for
	// establishing the transport connection.
	ProxyDial func(context.Context, string, string) (net.Conn, error)
}

func (s *DirectTCP) Handle(ctx context.Context) {
	var msg sshd.ChannelOpenDirectMsg
	if err := ssh.Unmarshal(s.NewChan.ExtraData(), &msg); err != nil {
		if s.Logger != nil {
			s.Logger.Println("unable to setup forwarding:", err)
		}
		s.NewChan.Reject(ssh.ResourceShortage, "Error parsing message")
		return
	}

	outbound, err := s.proxyDial(ctx, "tcp", fmt.Sprintf("%s:%d", msg.RAddr, msg.RPort))
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

	go ssh.DiscardRequests(reqs)
	err = tunnel(ctx, ch, outbound, buf1, buf2)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Println("tunnel:", err)
		}
		return
	}
}

func (s *DirectTCP) proxyDial(ctx context.Context, network, address string) (net.Conn, error) {
	proxyDial := s.ProxyDial
	if proxyDial == nil {
		var dialer net.Dialer
		proxyDial = dialer.DialContext
	}
	return proxyDial(ctx, network, address)
}

// tunnel create tunnels for two io.ReadWriteCloser
func tunnel(ctx context.Context, c1, c2 io.ReadWriteCloser, buf1, buf2 []byte) error {
	ctx, cancel := context.WithCancel(ctx)
	var errs tunnelErr
	go func() {
		_, errs[0] = io.CopyBuffer(c1, c2, buf1)
		cancel()
	}()
	go func() {
		_, errs[1] = io.CopyBuffer(c2, c1, buf2)
		cancel()
	}()
	<-ctx.Done()
	errs[2] = c1.Close()
	errs[3] = c2.Close()
	errs[4] = ctx.Err()
	if errs[4] == context.Canceled {
		errs[4] = nil
	}
	return errs.FirstError()
}

type tunnelErr [5]error

func (t tunnelErr) FirstError() error {
	for _, err := range t {
		if err != nil {
			return err
		}
	}
	return nil
}
