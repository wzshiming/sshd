package tcpforward

import (
	"context"
	"sync"

	"github.com/wzshiming/sshd"
	"golang.org/x/crypto/ssh"
)

func init() {
	sshd.RegistryHandleRequest("tcpip-forward", Forward)
	sshd.RegistryHandleRequest("cancel-tcpip-forward", Cancel)
}

var (
	mut     sync.Mutex
	cancels = map[uint32]context.CancelFunc{}
)

func cancelPort(port uint32) {
	if cancel, ok := cancels[port]; ok {
		cancel()
		delete(cancels, port)
	}
}

func Forward(ctx context.Context, req *ssh.Request, serverConn *sshd.ServerConn) {
	d := &TCPForward{
		ServerConn: serverConn,
		BytesPool:  serverConn.BytesPool,
		Logger:     serverConn.Logger,
	}
	mut.Lock()
	defer mut.Unlock()
	d.Forward(ctx, req)

}

func Cancel(ctx context.Context, req *ssh.Request, serverConn *sshd.ServerConn) {
	d := &TCPForward{
		ServerConn: serverConn,
		BytesPool:  serverConn.BytesPool,
		Logger:     serverConn.Logger,
	}
	mut.Lock()
	defer mut.Unlock()
	d.Cancel(ctx, req)
}
