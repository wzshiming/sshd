package directtcp

import (
	"context"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

func init() {
	sshd.RegistryHandle("direct-tcpip", Handle)
}

func Handle(ctx context.Context, newChan ssh.NewChannel, serverConn *sshd.ServerConn) {
	d := &DirectTCP{
		NewChan:   newChan,
		ProxyDial: serverConn.ProxyDial,
		BytesPool: serverConn.BytesPool,
		Logger:    serverConn.Logger,
	}
	d.Handle(ctx)
}
