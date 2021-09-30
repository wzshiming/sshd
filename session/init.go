package session

import (
	"context"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

func init() {
	sshd.RegistryHandle("session", Handle)
}

func Handle(ctx context.Context, newChan ssh.NewChannel, serverConn *sshd.ServerConn) {
	s := &Session{
		NewChan: newChan,
		Environ: serverConn.Environ,
		Dir:     serverConn.Dir,
		Logger:  serverConn.Logger,
	}
	s.Handle(ctx)
}
