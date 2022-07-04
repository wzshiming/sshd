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
	cancels map[uint32]io.Closer
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

		err = sshd.Tunnel(ctx, conn, chans, buf1, buf2)
		if err != nil && !sshd.IsClosedConnError(err) {
			if serverConn.Logger != nil {
				serverConn.Logger.Println("Tunnel:", err)
			}
		}
	}
}

func (s *TCPForward) Forward(ctx context.Context, req *ssh.Request, serverConn *sshd.ServerConn) {
	s.mut.Lock()
	defer s.mut.Unlock()
	m := sshd.ForwardMsg{}
	err := ssh.Unmarshal(req.Payload, &m)
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("Unmarshal:", err)
		}
		req.Reply(false, nil)
		return
	}

	s.cancelPort(m.LPort)

	k := fmt.Sprintf("%s:%d", m.LAddr, m.LPort)

	listener, err := s.proxyListen(ctx, serverConn, "tcp", k)
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

	s.setCancelPort(port, listener)
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
	s.mut.Lock()
	defer s.mut.Unlock()
	m := sshd.ForwardMsg{}
	err := ssh.Unmarshal(req.Payload, &m)
	if err != nil {
		if serverConn.Logger != nil {
			serverConn.Logger.Println("Unmarshal:", err)
		}
		req.Reply(false, nil)
		return
	}

	s.cancelPort(m.LPort)
	req.Reply(true, nil)
}

func (s *TCPForward) cancelPort(port uint32) {
	if s.cancels == nil {
		return
	}
	if cancel, ok := s.cancels[port]; ok {
		cancel.Close()
		delete(s.cancels, port)
	}
}

func (s *TCPForward) setCancelPort(port uint32, cf io.Closer) {
	if s.cancels == nil {
		s.cancels = map[uint32]io.Closer{}
	}
	if cancel, ok := s.cancels[port]; ok {
		cancel.Close()
	}
	s.cancels[port] = cf
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
