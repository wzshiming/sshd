package tcpforward

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

// TCPForward Handling for a single incoming connection
type TCPForward struct {
	*sshd.ServerConn
}

func (s *TCPForward) forwardListener(ctx context.Context, listener net.Listener, cancel func()) {
	defer cancel()
	_, port, err := ParseAddr(listener.Addr().String())
	if err != nil {
		if s.Logger != nil {
			s.Logger.Println("ParseAddr:", err)
		}
		return
	}

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

		ohost, oport, err := ParseAddr(conn.RemoteAddr().String())
		if err != nil {
			if s.Logger != nil {
				s.Logger.Println("ParseAddr:", err)
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
		chans, reqs, err := s.OpenChannel("forwarded-tcpip", data)
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
			return
		}
	}
}

func (s *TCPForward) Forward(ctx context.Context, req *ssh.Request) {
	m := sshd.ForwardMsg{}
	err := ssh.Unmarshal(req.Payload, &m)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Println("Unmarshal:", err)
		}
		req.Reply(false, nil)
		return
	}

	cancelPort(m.LPort)

	k := fmt.Sprintf("%s:%d", m.LAddr, m.LPort)

	listener, err := s.proxyListen(ctx, "tcp", k)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Println("Listen:", err)
		}
		req.Reply(false, nil)
		return
	}

	_, port, err := ParseAddr(listener.Addr().String())
	if err != nil {
		if s.Logger != nil {
			s.Logger.Println("ParseAddr:", err)
		}
		req.Reply(false, nil)
		return
	}

	cancels[port] = func() {
		listener.Close()
	}
	go s.forwardListener(ctx, listener, func() {
		cancelPort(port)
	})

	resp := ssh.Marshal(sshd.ForwardResponseMsg{
		Port: port,
	})
	req.Reply(true, resp)
}

func (s *TCPForward) proxyListen(ctx context.Context, network, address string) (net.Listener, error) {
	proxyListen := s.ProxyListen
	if proxyListen == nil {
		var listenConfig net.ListenConfig
		proxyListen = listenConfig.Listen
	}
	return proxyListen(ctx, network, address)
}

type TCPForwardCancel struct {
	*sshd.ServerConn
}

func (s *TCPForwardCancel) Cancel(ctx context.Context, req *ssh.Request) {
	m := sshd.ForwardMsg{}
	err := ssh.Unmarshal(req.Payload, &m)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Println("Unmarshal:", err)
		}
		req.Reply(false, nil)
		return
	}

	cancelPort(m.LPort)
	req.Reply(true, nil)
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
