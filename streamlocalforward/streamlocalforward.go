package streamlocalforward

import (
	"context"
	"net"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

// StreamLocalForward Handling for a single incoming connection
type StreamLocalForward struct {
	*sshd.ServerConn
}

func (s *StreamLocalForward) forwardListener(ctx context.Context, listener net.Listener, cancel func()) {
	defer cancel()

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

	for {
		conn, err := listener.Accept()
		if err != nil {
			if sshd.IsClosedConnError(err) {
				return
			}
			if s.Logger != nil {
				s.Logger.Println("listen Accept:", err)
			}
			return
		}

		resp := sshd.ForwardedStreamLocalPayload{
			SocketPath: listener.Addr().String(),
		}

		data := ssh.Marshal(resp)
		chans, reqs, err := s.OpenChannel("forwarded-streamlocal@openssh.com", data)
		if err != nil {
			if s.Logger != nil {
				s.Logger.Println("OpenChannel:", err)
			}
			return
		}

		go sshd.DiscardRequests(s.Logger, reqs)

		err = sshd.Tunnel(ctx, conn, chans, buf1, buf2)
		if err != nil && !sshd.IsClosedConnError(err) {
			if s.Logger != nil {
				s.Logger.Println("Tunnel:", err)
			}
		}
	}
}

func (s *StreamLocalForward) Forward(ctx context.Context, req *ssh.Request) {
	m := sshd.StreamLocalChannelForwardMsg{}
	err := ssh.Unmarshal(req.Payload, &m)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Println("Unmarshal:", err)
		}
		req.Reply(false, nil)
		return
	}

	cancelPath(m.SocketPath)

	listener, err := s.proxyListen(ctx, "unix", m.SocketPath)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Println("Listen:", err)
		}
		req.Reply(false, nil)
		return
	}

	setCancelPath(m.SocketPath, func() {
		listener.Close()
	})
	go s.forwardListener(ctx, listener, func() {
		cancelPath(m.SocketPath)
	})

	req.Reply(true, nil)
}

func (s *StreamLocalForward) proxyListen(ctx context.Context, network, address string) (net.Listener, error) {
	proxyListen := s.ProxyListen
	if proxyListen == nil {
		var listenConfig net.ListenConfig
		proxyListen = listenConfig.Listen
	}
	return proxyListen(ctx, network, address)
}

type StreamLocalForwardCancel struct {
	*sshd.ServerConn
}

func (s *StreamLocalForwardCancel) Cancel(ctx context.Context, req *ssh.Request) {
	m := sshd.StreamLocalChannelForwardMsg{}
	err := ssh.Unmarshal(req.Payload, &m)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Println("Unmarshal:", err)
		}
		req.Reply(false, nil)
		return
	}

	cancelPath(m.SocketPath)
	req.Reply(true, nil)
}
