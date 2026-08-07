package tcpforward

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

// TCPForward Handling for a single incoming connection
type TCPForward struct {
	mut     sync.Mutex
	cancels map[cancelKey]io.Closer
}

// cancelKey scopes a forwarded port to its connection so one
// connection cannot cancel another connection's forward.
type cancelKey struct {
	conn *sshd.ServerConn
	port uint32
}

func (s *TCPForward) forwardListener(ctx context.Context, serverConn *sshd.ServerConn, listener net.Listener) {
	defer listener.Close()
	_, port, err := ParseAddr(listener.Addr().String())
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("ParseAddr:", err)
		}
		return
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

		ohost, oport, err := ParseAddr(conn.RemoteAddr().String())
		if err != nil {
			if serverConn.Logger != nil {
				serverConn.Logger.Println("ParseAddr:", err)
			}
			return
		}
		resp := sshd.ForwardedTCPPayload{
			Addr:       "0.0.0.0",
			Port:       port,
			OriginAddr: ohost,
			OriginPort: oport,
		}

		data := ssh.Marshal(resp)
		chans, reqs, err := serverConn.OpenChannel("forwarded-tcpip", data)
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

func (s *TCPForward) tunnel(ctx context.Context, serverConn *sshd.ServerConn, conn net.Conn, chans ssh.Channel) {
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

func (s *TCPForward) Forward(ctx context.Context, req *ssh.Request, serverConn *sshd.ServerConn) {
	m := sshd.ForwardMsg{}
	err := ssh.Unmarshal(req.Payload, &m)
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("Unmarshal:", err)
		}
		req.Reply(false, nil)
		return
	}

	local := fmt.Sprintf("%s:%d", formatLocalAddr(m.LAddr), m.LPort)
	if serverConn.Permissions != nil && !serverConn.Permissions.Allow(name, local) {
		req.Reply(false, nil)
		return
	}

	s.cancelPort(serverConn, m.LPort)

	listener, err := s.proxyListen(ctx, serverConn, "tcp", local)
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("Listen:", err)
		}
		req.Reply(false, nil)
		return
	}

	_, port, err := ParseAddr(listener.Addr().String())
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("ParseAddr:", err)
		}
		req.Reply(false, nil)
		return
	}

	s.setCancelPort(serverConn, port, listener)
	go s.forwardListener(ctx, serverConn, listener)

	resp := ssh.Marshal(sshd.ForwardResponseMsg{
		Port: port,
	})
	req.Reply(true, resp)
}

func (s *TCPForward) proxyListen(ctx context.Context, serverConn *sshd.ServerConn, network, address string) (net.Listener, error) {
	proxyListen := serverConn.ProxyListen
	if proxyListen == nil {
		var listenConfig net.ListenConfig
		proxyListen = listenConfig.Listen
	}
	return proxyListen(ctx, network, address)
}

func (s *TCPForward) Cancel(ctx context.Context, req *ssh.Request, serverConn *sshd.ServerConn) {
	m := sshd.ForwardMsg{}
	err := ssh.Unmarshal(req.Payload, &m)
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("Unmarshal:", err)
		}
		req.Reply(false, nil)
		return
	}

	local := fmt.Sprintf("%s:%d", formatLocalAddr(m.LAddr), m.LPort)
	if serverConn.Permissions != nil && !serverConn.Permissions.Allow(name, local) {
		req.Reply(false, nil)
		return
	}

	s.cancelPort(serverConn, m.LPort)
	req.Reply(true, nil)
}

func (s *TCPForward) cancelPort(serverConn *sshd.ServerConn, port uint32) {
	s.mut.Lock()
	defer s.mut.Unlock()
	key := cancelKey{serverConn, port}
	if cancel, ok := s.cancels[key]; ok {
		cancel.Close()
		delete(s.cancels, key)
	}
}

func (s *TCPForward) setCancelPort(serverConn *sshd.ServerConn, port uint32, cf io.Closer) {
	s.mut.Lock()
	defer s.mut.Unlock()
	if s.cancels == nil {
		s.cancels = map[cancelKey]io.Closer{}
	}
	key := cancelKey{serverConn, port}
	if cancel, ok := s.cancels[key]; ok {
		cancel.Close()
	}
	s.cancels[key] = cf
}

func ParseAddr(addr string) (string, uint32, error) {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.ParseUint(p, 10, 32)
	if err != nil {
		return "", 0, err
	}
	return host, uint32(port), nil
}

func formatLocalAddr(addr string) string {
	if addr == "" {
		return ""
	}
	ip := net.ParseIP(addr)
	if ip == nil {
		return "127.0.0.1"
	}
	if ip.IsLoopback() {
		return ip.String()
	}
	return ""
}
