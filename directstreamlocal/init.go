package directstreamlocal

import (
	"context"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

func init() {
	sshd.RegistryHandleChannel("direct-streamlocal@openssh.com", Handle)
}

func Handle(ctx context.Context, newChan ssh.NewChannel, serverConn *sshd.ServerConn) {
	d := &DirectStreamLocal{
		NewChan:    newChan,
		ServerConn: serverConn,
	}
	d.Handle(ctx)
}
