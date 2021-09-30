package sshd

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net"
	"os"
	"os/user"

	"golang.org/x/crypto/ssh"
)

// Server is accepting connections and handling the details of the SSH protocol
type Server struct {
	// Context is default context
	Context context.Context
	// ServerConfig SSH Server config
	ServerConfig ssh.ServerConfig
	// Logger error log
	Logger Logger
	// ProxyDial specifies the optional proxyDial function for
	// establishing the transport connection.
	ProxyDial func(context.Context, string, string) (net.Conn, error)
	// BytesPool getting and returning temporary bytes for use by io.CopyBuffer
	BytesPool BytesPool
	// Default environment
	Environ []string
	// Default workdir
	Dir string
}

func NewServer() *Server {
	s := &Server{}
	s.Environ = os.Environ()
	if userInfo, err := user.Current(); err == nil {
		s.Dir = userInfo.HomeDir
	}
	return s
}

func (s *Server) context() context.Context {
	if s.Context == nil {
		return context.Background()
	}
	return s.Context
}

// ListenAndServe is used to create a listener and serve on it
func (s *Server) ListenAndServe(network, addr string) error {
	var lc net.ListenConfig
	l, err := lc.Listen(s.context(), network, addr)
	if err != nil {
		return err
	}
	return s.Serve(l)
}

// Serve is used to serve connections from a listener
func (s *Server) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go s.ServeConn(conn)
	}
}

// ServeConn is used to serve a single connection.
func (s *Server) ServeConn(conn net.Conn) {
	c, err := NewServerConn(conn, &s.ServerConfig)
	if err != nil {
		s.Logger.Println("unable to negotiate ssh:", err)
		return
	}
	defer c.Close()
	c.ProxyDial = s.ProxyDial
	c.Logger = s.Logger
	c.BytesPool = s.BytesPool
	c.Environ = s.Environ
	c.Dir = s.Dir
	c.Handle(s.context())
}

func (s *Server) AddHostkey(keyData []byte) error {
	key, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return err
	}
	s.ServerConfig.AddHostKey(key)
	return nil
}

func (s *Server) RandomHostkey() error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	signer, err := ssh.NewSignerFromSigner(key)
	if err != nil {
		return err
	}
	s.ServerConfig.AddHostKey(signer)
	return nil
}

func GetAuthorizedFile(authorized string) (map[string]string, error) {
	f, err := os.Open(authorized)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ParseAuthorized(f)
}

func ParseAuthorized(r io.Reader) (map[string]string, error) {
	keys := map[string]string{}
	read := bufio.NewReader(r)
	for {
		line, _, err := read.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if key, cmt, _, _, err := ssh.ParseAuthorizedKey(line); err == nil {
			keys[string(key.Marshal())] = cmt
		}
	}
	return keys, nil
}
