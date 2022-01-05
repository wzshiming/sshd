package directtcp

import (
	"context"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

func init() {
	sshd.RegistryHandleChannel("direct-tcpip", Handle)
}

func Handle(ctx context.Context, newChan ssh.NewChannel, serverConn *sshd.ServerConn) {
	d := &DirectTCP{
		NewChan:    newChan,
		ServerConn: serverConn,
	}
	d.Handle(ctx)
}
