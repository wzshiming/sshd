package streamlocalforward

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

// StreamLocalForward Handling for a single incoming connection
type StreamLocalForward struct {
	mut     sync.Mutex
	cancels map[string]io.Closer
}

func (s *StreamLocalForward) forwardListener(ctx context.Context, serverConn *sshd.ServerConn, listener net.Listener) {
	defer listener.Close()

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

	for {
		conn, err := listener.Accept()
		if err != nil {
			if sshd.IsClosedConnError(err) {
				return
			}
			if serverConn.Logger != nil {
				serverConn.Logger.Println("listen Accept:", err)
			}
			return
		}

		resp := sshd.ForwardedStreamLocalPayload{
			SocketPath: listener.Addr().String(),
		}

		data := ssh.Marshal(resp)
		chans, reqs, err := serverConn.OpenChannel("forwarded-streamlocal@openssh.com", data)
		if err != nil {
			if serverConn.Logger != nil {
				serverConn.Logger.Println("OpenChannel:", err)
			}
			return
		}

		go sshd.DiscardRequests(serverConn.Logger, reqs)

		err = sshd.Tunnel(ctx, conn, chans, buf1, buf2)
		if err != nil && !sshd.IsClosedConnError(err) {
			if serverConn.Logger != nil {
				serverConn.Logger.Println("Tunnel:", err)
			}
		}
	}
}

func (s *StreamLocalForward) Forward(ctx context.Context, req *ssh.Request, serverConn *sshd.ServerConn) {
	s.mut.Lock()
	defer s.mut.Unlock()
	m := sshd.StreamLocalChannelForwardMsg{}
	err := ssh.Unmarshal(req.Payload, &m)
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("Unmarshal:", err)
		}
		req.Reply(false, nil)
		return
	}

	if serverConn.Permissions != nil && !serverConn.Permissions.Allow(name, m.SocketPath) {
		req.Reply(false, nil)
		return
	}

	s.cancelPath(m.SocketPath)

	listener, err := s.proxyListen(ctx, serverConn, "unix", m.SocketPath)
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("Listen:", err)
		}
		req.Reply(false, nil)
		return
	}

	s.setCancelPath(m.SocketPath, listener)
	go s.forwardListener(ctx, serverConn, listener)

	req.Reply(true, nil)
}

func (s *StreamLocalForward) proxyListen(ctx context.Context, serverConn *sshd.ServerConn, network, address string) (net.Listener, error) {
	proxyListen := serverConn.ProxyListen
	if proxyListen == nil {
		var listenConfig net.ListenConfig
		proxyListen = listenConfig.Listen
	}
	return proxyListen(ctx, network, address)
}

func (s *StreamLocalForward) Cancel(ctx context.Context, req *ssh.Request, serverConn *sshd.ServerConn) {
	s.mut.Lock()
	defer s.mut.Unlock()
	m := sshd.StreamLocalChannelForwardMsg{}
	err := ssh.Unmarshal(req.Payload, &m)
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("Unmarshal:", err)
		}
		req.Reply(false, nil)
		return
	}

	if serverConn.Permissions != nil && !serverConn.Permissions.Allow(name, m.SocketPath) {
		req.Reply(false, nil)
		return
	}

	s.cancelPath(m.SocketPath)
	req.Reply(true, nil)
}

func (s *StreamLocalForward) cancelPath(path string) {
	if s.cancels == nil {
		return
	}
	if cancel, ok := s.cancels[path]; ok {
		cancel.Close()
		delete(s.cancels, path)
	}
}

func (s *StreamLocalForward) setCancelPath(path string, cf io.Closer) {
	if s.cancels == nil {
		s.cancels = map[string]io.Closer{}
	}
	if cancel, ok := s.cancels[path]; ok {
		cancel.Close()
	}
	s.cancels[path] = cf
}
