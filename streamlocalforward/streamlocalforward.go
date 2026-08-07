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
	cancels map[cancelKey]io.Closer
}

// cancelKey scopes a forwarded socket path to its connection so one
// connection cannot cancel another connection's forward.
type cancelKey struct {
	conn *sshd.ServerConn
	path string
}

func (s *StreamLocalForward) forwardListener(ctx context.Context, serverConn *sshd.ServerConn, listener net.Listener) {
	defer listener.Close()

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

		go s.tunnel(ctx, serverConn, conn, chans)
	}
}

func (s *StreamLocalForward) tunnel(ctx context.Context, serverConn *sshd.ServerConn, conn net.Conn, chans ssh.Channel) {
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
	err := sshd.Tunnel(ctx, conn, chans, buf1, buf2)
	if err != nil && !sshd.IsClosedConnError(err) {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("Tunnel:", err)
		}
	}
}

func (s *StreamLocalForward) Forward(ctx context.Context, req *ssh.Request, serverConn *sshd.ServerConn) {
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

	s.cancelPath(serverConn, m.SocketPath)

	listener, err := s.proxyListen(ctx, serverConn, "unix", m.SocketPath)
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("Listen:", err)
		}
		req.Reply(false, nil)
		return
	}

	s.setCancelPath(serverConn, m.SocketPath, listener)
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

	s.cancelPath(serverConn, m.SocketPath)
	req.Reply(true, nil)
}

func (s *StreamLocalForward) cancelPath(serverConn *sshd.ServerConn, path string) {
	s.mut.Lock()
	defer s.mut.Unlock()
	key := cancelKey{serverConn, path}
	if cancel, ok := s.cancels[key]; ok {
		cancel.Close()
		delete(s.cancels, key)
	}
}

func (s *StreamLocalForward) setCancelPath(serverConn *sshd.ServerConn, path string, cf io.Closer) {
	s.mut.Lock()
	defer s.mut.Unlock()
	if s.cancels == nil {
		s.cancels = map[cancelKey]io.Closer{}
	}
	key := cancelKey{serverConn, path}
	if cancel, ok := s.cancels[key]; ok {
		cancel.Close()
	}
	s.cancels[key] = cf
}
