package sshd

import (
	"context"
	"net"

	"golang.org/x/crypto/ssh"
)

type HandleChannelFunc func(ctx context.Context, newChan ssh.NewChannel, serverConn *ServerConn)

var registryChannel = map[string]HandleChannelFunc{}

func RegistryHandleChannel(name string, fun HandleChannelFunc) {
	registryChannel[name] = fun
}

type HandleRequestFunc func(ctx context.Context, req *ssh.Request, serverConn *ServerConn)

var registryRequest = map[string]HandleRequestFunc{}

func RegistryHandleRequest(name string, fun HandleRequestFunc) {
	registryRequest[name] = fun
}

// ServerConn Handling for a single incoming connection
type ServerConn struct {
	*ssh.ServerConn
	// BytesPool getting and returning temporary bytes for use by io.CopyBuffer
	BytesPool BytesPool
	// Logger error log
	Logger Logger
	// Newly Request
	Requests <-chan *ssh.Request
	// Newly channel
	Channels <-chan ssh.NewChannel
	// ProxyDial specifies the optional proxyDial function for
	// establishing the transport connection.
	ProxyDial func(context.Context, string, string) (net.Conn, error)
	// ProxyListen specifies the optional proxyListen function for
	// establishing the transport connection.
	ProxyListen func(context.Context, string, string) (net.Listener, error)
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
	go s.handleRequests(ctx)
	s.handleChannels(ctx)
}

func (s *ServerConn) handleRequests(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-s.Requests:
			if !ok {
				return
			}
			if handle, ok := registryRequest[req.Type]; ok && handle != nil {
				handle(ctx, req, s)
				continue
			} else if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

func (s *ServerConn) handleChannels(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case newChan, ok := <-s.Channels:
			if !ok {
				return
			}
			chType := newChan.ChannelType()
			channel, ok := registryChannel[chType]
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

// DiscardRequests consumes and rejects all requests from the
// passed-in channel.
func DiscardRequests(logger Logger, in <-chan *ssh.Request) {
	for req := range in {
		if logger != nil {
			logger.Println("Ignore Requests", req.Type)
		}
		if req.WantReply {
			req.Reply(false, nil)
		}
	}
}
