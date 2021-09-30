package sshd

import (
	"context"
	"net"

	"golang.org/x/crypto/ssh"
)

type HandleFunc func(ctx context.Context, newChan ssh.NewChannel, serverConn *ServerConn)

var registry = map[string]HandleFunc{}

func RegistryHandle(name string, fun HandleFunc) {
	registry[name] = fun
}

// ServerConn Handling for a single incoming connection
type ServerConn struct {
	*ssh.ServerConn
	// BytesPool getting and returning temporary bytes for use by io.CopyBuffer
	BytesPool BytesPool
	// Logger error log
	Logger Logger
	// ignore
	Requests <-chan *ssh.Request
	// Newly channel
	Channels <-chan ssh.NewChannel
	// ProxyDial specifies the optional proxyDial function for
	// establishing the transport connection.
	ProxyDial func(context.Context, string, string) (net.Conn, error)
	// Default environment
	Environ []string
	// Default workdir
	Dir string
}

func NewServerConn(conn net.Conn, config *ssh.ServerConfig) (*ServerConn, error) {
	serverConn, channels, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return nil, err
	}
	return &ServerConn{
		ServerConn: serverConn,
		Requests:   requests,
		Channels:   channels,
	}, nil
}

// Handle a single established connection
func (s *ServerConn) Handle(ctx context.Context) {
	go ssh.DiscardRequests(s.Requests)

	for {
		select {
		case <-ctx.Done():
			return
		case newChan, ok := <-s.Channels:
			if !ok {
				return
			}
			chType := newChan.ChannelType()
			channel, ok := registry[chType]
			if ok && channel != nil {
				go channel(ctx, newChan, s)
			} else {
				if s.Logger != nil {
					s.Logger.Println("unknown channel type:", chType)
				}
				newChan.Reject(ssh.Prohibited, "Prohibited")
			}
		}
	}
}
